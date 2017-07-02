BASEDIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GOOS ?= linux
GOARCH ?= amd64

build:
	docker run --rm -v "$(BASEDIR)":/usr/src/mqtt2influx -w /usr/src/mqtt2influx golang:alpine \
		sh -c "apk add --update --no-cache git openssh && \
		go get -d ./... && \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -a -installsuffix cgo -o ./docker/mqtt2influx mqtt2influx.go"
docker: build
	cp config.toml ./docker/config.toml
	docker build -t nonsenz/mqtt2influx ./docker/
	$(MAKE) clean
clean:
	rm -f ./docker/mqtt2influx
	rm -f ./docker/config.toml

