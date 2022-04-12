GOFILES := $(shell find . -name '*.go')
GOFLAG := -gcflags=-G=3
GOBUILD := go build -v
binary := bin/rpcmux bin/rpcmux-playbook

build: ${binary}

all: test build

bin/rpcmux: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/server/main.go

bin/rpcmux-playbook: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/playbook/main.go

test:
	go test -v ./...

clean:
	rm -rf build dist ${binary}

golint:
	go fmt ./...
	go vet ./...

.PHONY: build all test govet gofmt dist
