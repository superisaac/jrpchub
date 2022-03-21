GOFILES := $(shell find . -name '*.go')
GOFLAG := -gcflags=-G=3
GOBUILD := GO111MODULE=on go build -v

build: bin/rpcz

all: test build

bin/rpcz: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/server/main.go

test:
	go test -v ./...

clean:
	rm -rf build dist bin/rpcz

golint:
	go fmt ./...
	go vet ./...

.PHONY: build all test govet gofmt dist
