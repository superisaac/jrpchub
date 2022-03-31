GOFILES := $(shell find . -name '*.go')
GOFLAG := -gcflags=-G=3
GOBUILD := go build -v

build: bin/rpcmap bin/rpcmap-playbook

all: test build

bin/rpcmap: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/server/main.go

bin/rpcmap-playbook: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/playbook/main.go

test:
	go test -v ./...

clean:
	rm -rf build dist bin/rpcmap bin/rpcmap-playbook

golint:
	go fmt ./...
	go vet ./...

.PHONY: build all test govet gofmt dist
