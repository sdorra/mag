setup:
	@go get github.com/codegangsta/negroni
	@go get github.com/gorilla/mux
	@go get github.com/vulcand/oxy/cbreaker
	@go get github.com/vulcand/oxy/forward
	@go get github.com/vulcand/oxy/roundrobin
	@go get github.com/vulcand/oxy/stream
	@go get github.com/Sirupsen/logrus
	@go get github.com/meatballhat/negroni-logrus
	@go get github.com/twinj/uuid
	@go get github.com/pkg/errors
