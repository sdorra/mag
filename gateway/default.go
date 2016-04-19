package gateway

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/mailgun/manners"
	"github.com/meatballhat/negroni-logrus"
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
		return nil, err
	}

	lb, err := roundrobin.New(fwd)
	if err != nil {
		return nil, err
	}

	stream, err := stream.New(lb, stream.Retry(`IsNetworkError() && Attempts() < 2`))
	if err != nil {
		return nil, err
	}

	circuitBreaker, err := cbreaker.New(stream, "NetworkErrorRatio() > 0.5")
	if err != nil {
		return nil, err
	}

	for _, url := range urls {
		log.Infoln("register new backend for path", path, url)
		err = lb.UpsertServer(url)
		if err != nil {
			return nil, err
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
func (ds *DefaultServer) ConfigureProxyRoutes(routes map[string][]*url.URL) error {
	log.Debugln("configure proxy routes")

	// handle new and update
	for path, urls := range routes {
		lb := ds.proxyRoutes[path]
		if lb != nil {
			err := ds.updateProxyRoute(path, lb, urls)
			if err != nil {
				return err
			}
		} else {
			lb, err := ds.addProxyRoute(path, urls)
			if err != nil {
				return err
			}
			ds.proxyRoutes[path] = lb
		}
	}

	// handle remove
	for path, lb := range ds.proxyRoutes {
		if routes[path] == nil {
			// Remove route completly ?
			ds.updateProxyRoute(path, lb, []*url.URL{})
		}
	}

	return nil
}

type status struct {
	Path     string
	Backends []string
}

// StatusHandler returns a http handler function which writes an json array for
// the current configured proxy routes.
func (ds *DefaultServer) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		states := []status{}
		for path, lb := range ds.proxyRoutes {
			backends := []string{}
			for _, url := range lb.Servers() {
				backends = append(backends, url.String())
			}
			states = append(states, status{path, backends})
		}
		json, err := json.Marshal(states)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

// Start will start the default gateway server. After the server is started the
// ConfigureProxyRoutes can be used to reconfigure the gateway.
func (ds *DefaultServer) Start() error {
	log.Infoln("starting gateway server")
	return ds.server.ListenAndServe()
}
