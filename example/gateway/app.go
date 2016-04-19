package main

import (
	"flag"

	"github.com/gorilla/mux"
	"github.com/sdorra/mag/discovery"
	"github.com/sdorra/mag/gateway"

	log "github.com/Sirupsen/logrus"
)

func watcher(server gateway.Server) discovery.Watcher {
	return func(services []*discovery.Service) error {
		proxyRoutes := []*gateway.ProxyRoute{}
		for _, service := range services {
			proxyRoutes = append(proxyRoutes, &gateway.ProxyRoute{
				Path:     "/" + service.Name,
				Backends: service.Backends,
			})
		}
		return server.ConfigureProxyRoutes(proxyRoutes)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)

	var url string
	flag.StringVar(&url, "consul", "consul://consul:8500", "url to consul")
	flag.Parse()

	discovery, err := discovery.NewConsulServiceDiscovery(url)
	if err != nil {
		log.WithError(err).Fatalf("failed to create service discovery")
	}

	router := mux.NewRouter()
	gs := gateway.NewDefaultServer(":8080", router)
	router.Handle("/status", gs.StatusHandler())
	discovery.Watch(watcher(gs))

	err = gs.Start()
	if err != nil {
		log.WithError(err).Fatalf("could not start server")
	}
}
