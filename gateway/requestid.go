package gateway

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/twinj/uuid"
)

// DefaultHeader contains the default http header which is used to store
// the request id
const DefaultHeader = "X-Request-Id"

// RequestIDGenerator is the functions used to generate the unique request id
type RequestIDGenerator func() (string, error)

// UUIDRequestIDGenerator is the default request id generator, which creates
// uuids
func UUIDRequestIDGenerator() (string, error) {
	return uuid.NewV4().String(), nil
}

// RequestID is negroni middleware which adds a unique request id to a specific
// http header of the request
type RequestID struct {
	Header   string
	Generate RequestIDGenerator
}

// NewRequestID creates a new request id middleware with a default generator
func NewRequestID() *RequestID {
	return &RequestID{
		Header:   DefaultHeader,
		Generate: UUIDRequestIDGenerator,
	}
}

// ServeHTTP adds the request id to the request and to the ResponseWriter
func (rid *RequestID) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	id, err := rid.Generate()
	log.Debug("append request id ", id)
	if err == nil {
		r.Header.Set(rid.Header, id)
		rw.Header().Set(rid.Header, id)
	} else {
		log.Warn("failed to create request id")
	}

	next(rw, r)
}
