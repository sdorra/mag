package gateway

import (
	"encoding/json"
	"net/url"
)

// ProxyRoute struct defines a route to a backend service.
type ProxyRoute struct {
	Path     string
	Backends []*url.URL
}

// MarshalJSON is used to marshal the ProxyRoute struct to a json object.
func (route *ProxyRoute) MarshalJSON() ([]byte, error) {
	backends := []string{}
	for _, url := range route.Backends {
		backends = append(backends, url.String())
	}
	return json.Marshal(map[string]interface{}{
		"path":     route.Path,
		"backends": backends,
	})
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
func ContainsRoute(routes []*ProxyRoute, path string) bool {
	for _, r := range routes {
		if r.Path == path {
			return true
		}
	}
	return false
}
