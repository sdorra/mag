package main

import (
	"flag"
	"log"
	"net"
	"strconv"

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
		log.Fatal(err)
	}

	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}

	id := createID()
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})
	r.GET("/"+serviceName, func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":     id,
			"name":   serviceName,
			"health": "ok",
		})
	})
	r.GET("/", func(c *gin.Context) {
		c.String(200, id)
	})

	log.Println("register service with consul", serviceName)
	_, err = registry.Register(discovery.ServiceRegistrationRequest{
		ID:              id,
		Name:            serviceName,
		Port:            port,
		HealthCheckPath: "/health",
	})

	if err != nil {
		log.Fatal(err)
	}

	defer registry.Unregister(id)

	r.Run(":" + strconv.Itoa(port))
}
