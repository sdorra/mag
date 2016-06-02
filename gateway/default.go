package gateway

import (
	"net/http"
	"net/url"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/mailgun/manners"
	"github.com/meatballhat/negroni-logrus"
	"github.com/pkg/errors"
	"github.com/vulcand/oxy/cbreaker"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/roundrobin"
	"github.com/vulcand/oxy/stream"

	log "github.com/Sirupsen/logrus"
)

// DefaultServer is the default gateway server implementation
type DefaultServer struct {
	server      *manners.GracefulServer
	router      *mux.Router
	proxyRoutes map[string]*roundrobin.RoundRobin
	middleware  []negroni.Handler
}

// NewDefaultServer creates a new DefaultServer. If the router parameter is nil
// the method will create a new router. It the middleware parameter is nil the
// method will use a logger and a recovery middleware.
func NewDefaultServer(addr string, router *mux.Router, middleware ...negroni.Handler) *DefaultServer {
	if router == nil {
		router = mux.NewRouter()
	}

	if len(middleware) <= 0 {
		middleware = append(middleware, negronilogrus.NewMiddleware())
		middleware = append(middleware, negroni.NewRecovery())
	}

	server := manners.NewWithServer(&http.Server{
		Addr:    addr,
		Handler: router,
	})

	log.Debugln("creating new gateway server for", addr)
	return &DefaultServer{server, router, map[string]*roundrobin.RoundRobin{}, middleware}
}

func (ds *DefaultServer) updateProxyRoute(path string, lb *roundrobin.RoundRobin, urls []*url.URL) error {
	log.Debugln("update proxy route", path)
	servers := lb.Servers()
	for _, url := range urls {
		if !ContainsURL(servers, url) {
			log.Infoln("register new backend", url)
			lb.UpsertServer(url)
		}
	}
	for _, url := range servers {
		if !ContainsURL(urls, url) {
			log.Infoln("unregister backend", url)
			lb.RemoveServer(url)
		}
	}
	return nil
}

func (ds *DefaultServer) addProxyRoute(path string, urls []*url.URL) (*roundrobin.RoundRobin, error) {
	log.Debugln("add proxy route", path)
	fwd, err := forward.New()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create forward")
	}

	lb, err := roundrobin.New(fwd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create roundrobin load balancer")
	}

	stream, err := stream.New(lb, stream.Retry(`IsNetworkError() && Attempts() < 2`))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stream")
	}

	circuitBreaker, err := cbreaker.New(stream, "NetworkErrorRatio() > 0.5")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create circuit breaker")
	}

	for _, url := range urls {
		log.Infoln("register new backend for path", path, url)
		err = lb.UpsertServer(url)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create upsert server for %s", url.String())
		}
	}

	// configure middleware for proxy backend
	middleware := negroni.New(ds.middleware...)
	middleware.UseHandler(circuitBreaker)
	ds.router.Handle(path, middleware)

	return lb, nil
}

// ConfigureProxyRoutes configures proxy routes. The map parameter must use the
// path for the route as key and the value must be a slice with urls for the
// backends. The method will configure a roundrobin load balancer for each path
// in the map.
func (ds *DefaultServer) ConfigureProxyRoutes(routes []*ProxyRoute) error {
	log.Debugln("configure proxy routes")

	// handle new and update
	for _, route := range routes {
		lb := ds.proxyRoutes[route.Path]
		if lb != nil {
			err := ds.updateProxyRoute(route.Path, lb, route.Backends)
			if err != nil {
				return errors.Wrapf(err, "failed to update proxy route for %s", route.Path)
			}
		} else {
			lb, err := ds.addProxyRoute(route.Path, route.Backends)
			if err != nil {
				return errors.Wrapf(err, "failed to add proxy route for %s", route.Path)
			}
			ds.proxyRoutes[route.Path] = lb
		}
	}

	// handle remove
	for path, lb := range ds.proxyRoutes {
		if !ContainsRoute(routes, path) {
			// Remove route completly ?
			ds.updateProxyRoute(path, lb, []*url.URL{})
		}
	}

	return nil
}

// GetProxyRoutes returns a slice of current configured proxy routes.
func (ds *DefaultServer) GetProxyRoutes() []*ProxyRoute {
	routes := []*ProxyRoute{}
	for path, lb := range ds.proxyRoutes {
		backends := []*url.URL{}
		for _, url := range lb.Servers() {
			backends = append(backends, url)
		}
		routes = append(routes, &ProxyRoute{path, backends})
	}
	return routes
}

// Start will start the default gateway server. After the server is started the
// ConfigureProxyRoutes can be used to reconfigure the gateway.
func (ds *DefaultServer) Start() error {
	log.Infoln("starting gateway server")
	return ds.server.ListenAndServe()
}
