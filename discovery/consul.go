package discovery

import (
	"net/url"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

// ConsulServiceRegistry is an implementation of the ServiceDiscovery interface
// which uses consul as service discovery registry.
type ConsulServiceRegistry struct {
	client *consulapi.Client
}

func (csr *ConsulServiceRegistry) Register(id string, name string, ip string, port int) error {
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = id
	registration.Name = name
	registration.Port = port
	registration.Tags = []string{"mag"}
	registration.Address = ip
	registration.Check = new(consulapi.AgentServiceCheck)
	registration.Check.HTTP = "http://" + ip + ":" + strconv.Itoa(port) + "/health"
	registration.Check.Interval = "30s"
	registration.Check.Status = "passing"
	// registration.Check.TTL = "30s"

	return csr.client.Agent().ServiceRegister(registration)
}

func (csr *ConsulServiceRegistry) Unregister(id string) error {
	return csr.client.Agent().ServiceDeregister(id)
}

func (csr *ConsulServiceRegistry) getBackends(service string) ([]*url.URL, error) {
	opts := &consulapi.QueryOptions{}
	services, _, err := csr.client.Health().Service(service, "mag", true, opts)
	if err != nil {
		return nil, err
	}

	backends := []*url.URL{}
	for _, value := range services {
		url, err := createURL(value.Service)
		if err != nil {
			return nil, err
		}
		backends = append(backends, url)
	}

	return backends, nil
}

func createURL(service *consulapi.AgentService) (*url.URL, error) {
	url, err := url.Parse("http://" + service.Address + ":" + strconv.Itoa(service.Port))
	if err != nil {
		return nil, err
	}
	return url, nil
}

func (csr *ConsulServiceRegistry) registerServices(watcher Watcher, services map[string][]string) {
	servicemap := map[string][]*url.URL{}
	for name, tags := range services {
		if ContainsString(tags, "mag") {
			urls, err := csr.getBackends(name)
			if err != nil {
				log.Warningln("could not retrieve backends for", name)
			} else {
				servicemap[name] = urls
			}
		}
	}
	watcher(servicemap)
}

// Watch registeres an Watcher for service changes.
func (csr *ConsulServiceRegistry) Watch(watcher Watcher) {
	opts := &consulapi.QueryOptions{WaitTime: 15 * time.Second}
	go func() {
		for {
			services, meta, err := csr.client.Catalog().Services(opts)
			if err != nil {
				log.Warningln("failed to list services", err)
			}

			if services != nil {
				csr.registerServices(watcher, services)
			}

			// If LastIndex didn't change then it means `Get` returned
			// because of the WaitTime and the key didn't changed.
			if opts.WaitIndex == meta.LastIndex {
				continue
			}
			opts.WaitIndex = meta.LastIndex
		}
	}()
}

// NewConsulServiceDiscovery creates an new service discovery for the consul
// backend.
func NewConsulServiceDiscovery(uri string) (*ConsulServiceRegistry, error) {
	log.Infoln("connecting to consul at", uri)
	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	config := consulapi.DefaultConfig()
	config.Address = url.Host
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &ConsulServiceRegistry{client}, err
}
