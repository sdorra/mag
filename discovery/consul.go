package discovery

import (
	"errors"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/twinj/uuid"
)

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

// ConsulServiceRegistry is an implementation of the ServiceDiscovery interface
// which uses consul as service discovery registry.
type ConsulServiceRegistry struct {
	client *consulapi.Client
}

// Register registers a service to the backend.
func (csr *ConsulServiceRegistry) Register(request ServiceRegistrationRequest) (string, error) {
	if request.ID == "" {
		request.ID = csr.createID()
	}
	if request.Name == "" {
		return "", errors.New("service name is required")
	}
	if request.Port == 0 {
		return "", errors.New("service port is required")
	}
	if request.Address == "" {
		var err error
		request.Address, err = csr.getLocalIP()
		if err != nil {
			return "", nil
		}
	}

	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = request.ID
	registration.Name = request.Name
	registration.Port = request.Port
	registration.Tags = []string{"mag"}
	registration.Address = request.Address

	if request.HealthCheckPath != "" {
		registration.Check = new(consulapi.AgentServiceCheck)
		registration.Check.HTTP = csr.createHealthURL(request)
		registration.Check.Interval = "30s"
		registration.Check.Status = "passing"
		// registration.Check.TTL = "30s"
	}

	err := csr.client.Agent().ServiceRegister(registration)
	if err != nil {
		return "", err
	}

	if request.EnableShutdownHook {
		csr.enableShutdownHook(request.ID)
	}

	return request.ID, nil
}

// Unregister removes the service with the given id from the backend.
func (csr *ConsulServiceRegistry) Unregister(id string) error {
	return csr.client.Agent().ServiceDeregister(id)
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

func (csr *ConsulServiceRegistry) enableShutdownHook(id string) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		csr.Unregister(id)
		os.Exit(0)
	}()
}

func (csr *ConsulServiceRegistry) createHealthURL(srr ServiceRegistrationRequest) string {
	return "http://" + srr.Address + ":" + strconv.Itoa(srr.Port) + srr.HealthCheckPath
}

func (csr *ConsulServiceRegistry) getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("could not resolve local ip address")
}

func (csr *ConsulServiceRegistry) createID() string {
	return uuid.NewV4().String()
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
