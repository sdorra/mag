package discovery

import "net/url"

// Watcher is notified when ever a service has changed. The watcher becomes
// always the full set of currently configured and healthy services.
type Watcher func(services map[string][]*url.URL) error

// ServiceRegistrationRequest struct is used to register a service.
type ServiceRegistrationRequest struct {
	ID                 string
	Name               string
	Address            string
	Tags               []string
	Port               int
	TTL                int
	EnableShutdownHook bool
}

// ServiceDiscovery contains methods for handling service discovery related
// operations.
type ServiceDiscovery interface {
	Watch(Watcher)
	Register(request ServiceRegistrationRequest) (string, error)
	Unregister(string) error
	Close()
}

// utils methods

// ContainsString is a util methods to check if an string is a member of a slice
// of strings.
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
