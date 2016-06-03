package gateway

import (
	"encoding/json"
	"net/url"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

// RouteBuilder configures a new sub route of router
type RouteBuilder func(router *mux.Router) (*mux.Route, error)

// ProxyRoute struct defines a route to a backend service.
type ProxyRoute struct {
	Name     string
	Backends []*url.URL
	Create   RouteBuilder
}

// MarshalJSON is used to marshal the ProxyRoute struct to a json object.
func (route *ProxyRoute) MarshalJSON() ([]byte, error) {
	backends := []string{}
	for _, url := range route.Backends {
		backends = append(backends, url.String())
	}
	return json.Marshal(map[string]interface{}{
		"name":     route.Name,
		"backends": backends,
	})
}

// ServerConfiguration is used to configure a new gateway server
type ServerConfiguration struct {
	Address    string
	CertFile   string
	KeyFile    string
	Router     *mux.Router
	Middleware []negroni.Handler
}

// Server is the gateway server which can be used to configure routes to backend
// services.
type Server interface {
	Start() error
	GetProxyRoutes() []*ProxyRoute
	ConfigureProxyRoutes([]*ProxyRoute) error
}

// util methods

// ContainsURL is a util methods to check if an url is a member of a slice of
// urls.
func ContainsURL(slice []*url.URL, url *url.URL) bool {
	urlString := url.String()
	for _, u := range slice {
		if u.String() == urlString {
			return true
		}
	}
	return false
}

// ContainsRoute is a util methods to check if an slice of proxy routes contains
// a route with the given path.
func ContainsRoute(routes []*ProxyRoute, name string) bool {
	for _, r := range routes {
		if r.Name == name {
			return true
		}
	}
	return false
}
