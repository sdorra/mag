package gateway

import (
	"net/http"
	"net/url"
)

// Server is the gateway server which can be used to configure routes to backend
// services.
type Server interface {
	Start() error
	StatusHandler() http.HandlerFunc
	ConfigureProxyRoutes(map[string][]*url.URL) error
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
