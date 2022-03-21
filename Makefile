GOFILES := $(shell find . -name '*.go')
GOFLAG := -gcflags=-G=3
GOBUILD := GO111MODULE=on go build -v

build: bin/rpcz

all: test build

bin/rpcz: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ server/http_server.go

test:
	go test -v ./...

clean:
	rm -rf build dist bin/rpcz

govet:
	go vet ./...

gofmt:
	go fmt ./...

.PHONY: build all test govet gofmt dist
