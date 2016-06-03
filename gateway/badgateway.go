package gateway

import (
	"net/http"

	"github.com/vulcand/oxy/roundrobin"
)

// BadGateway is a negroni middleware which returns status code 502 if the load
// balancer has no backend route
type BadGateway struct {
	lb *roundrobin.RoundRobin
}

// ServeHTTP returns status code 502, if the load balancer has no configured
// backend
func (bg *BadGateway) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if len(bg.lb.Servers()) == 0 {
		http.Error(rw, "502 Bad Gateway", http.StatusBadGateway)
	} else {
		next(rw, r)
	}
}
