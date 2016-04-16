package main

import (
	"errors"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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

func getLocalIP() (string, error) {
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

func createId() string {
	return uuid.NewV4().String()
}

func main() {
	serviceName := os.Args[1]
	if serviceName == "" {
		serviceName = "service"
	}

	url, err := url.Parse("consul://consul:8500")
	if err != nil {
		log.Fatal(err)
	}

	discovery, err := discovery.NewConsulServiceDiscovery(url)
	if err != nil {
		log.Fatal(err)
	}

	ip, err := getLocalIP()
	if err != nil {
		log.Fatal(err)
	}

	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}

	id := createId()
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigc
		discovery.Unregister(id)
		os.Exit(1)
	}()

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

	go discovery.Register(id, serviceName, ip, port)
	defer discovery.Unregister(id)

	r.Run(":" + strconv.Itoa(port))
}
