package gateway

import (
	"net/http"
	"net/url"
)

// ProxyRoute struct defines a route to a backend service.
type ProxyRoute struct {
	Path     string
	Backends []*url.URL
}

// Server is the gateway server which can be used to configure routes to backend
// services.
type Server interface {
	Start() error
	StatusHandler() http.HandlerFunc
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
func ContainsRoute(routes []*ProxyRoute, path string) bool {
	for _, r := range routes {
		if r.Path == path {
			return true
		}
	}
	return false
}
