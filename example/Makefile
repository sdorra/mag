DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
	
build:
	@cd $(DIR)/gateway; GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w' -o $(DIR)/dist/gateway
	@cd $(DIR)/service; GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w' -o $(DIR)/dist/service
