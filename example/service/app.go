package main

import (
	"flag"
	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/sdorra/mag/discovery"
	"github.com/twinj/uuid"
)

func getFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func createID() string {
	return uuid.NewV4().String()
}

func main() {
	var url string
	flag.StringVar(&url, "consul", "consul://consul:8500", "url to consul")
	var serviceName string
	flag.StringVar(&serviceName, "service", "sample", "name for the service")
	flag.Parse()

	registry, err := discovery.NewConsulServiceDiscovery(url)
	if err != nil {
		log.WithError(err).Fatalln("could not create service discovery")
	}

	port, err := getFreePort()
	if err != nil {
		log.WithError(err).Fatalln("could not get free port")
	}

	id := createID()

	r := gin.New()
	r.GET("/"+serviceName, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":      id,
			"name":    serviceName,
			"health":  "ok",
			"request": c.Request.Header["X-Request-Id"],
		})
	})

	r.GET("/", func(c *gin.Context) {
		c.String(200, id)
	})

	log.Println("register service with consul", serviceName)
	_, err = registry.Register(discovery.ServiceRegistrationRequest{
		ID:                 id,
		Name:               serviceName,
		Port:               port,
		TTL:                10,
		EnableShutdownHook: true,
	})

	if err != nil {
		log.WithError(err).Fatalln("could not register service")
	}

	defer registry.Unregister(id)
	defer registry.Close()
	r.Run(":" + strconv.Itoa(port))
}
