package main

import (
	"log"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/sdorra/mag/discovery"
	"github.com/sdorra/mag/gateway"
)

func watcher(server gateway.Server) discovery.Watcher {
	return func(services map[string][]*url.URL) error {
		proxyRoutes := map[string][]*url.URL{}
		for name, urls := range services {
			proxyRoutes["/"+name] = urls
		}
		return server.ConfigureProxyRoutes(proxyRoutes)
	}
}

func main() {
	url, err := url.Parse("consul://consul:8500")
	if err != nil {
		log.Fatal(err)
	}

	discovery, err := discovery.NewConsulServiceDiscovery(url)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	gs := gateway.NewDefaultServer(":8080", router, nil)
	router.Handle("/status", gs.StatusHandler())
	discovery.Watch(watcher(gs))
	gs.Start()
}
