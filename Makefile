GOFILES := $(shell find . -name '*.go')
GOFLAG := -gcflags=-G=3
GOBUILD := go build -v
binary := bin/jrpchub bin/jrpchub-playbook

build: ${binary}

all: test build

bin/jrpchub: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/server/main.go

bin/jrpchub-playbook: ${GOFILES}
	${GOBUILD} ${GOFLAG} -o $@ cmd/playbook/main.go

test:
	go test -v ./...

clean:
	rm -rf build dist ${binary}

golint:
	go fmt ./...
	go vet ./...

.PHONY: build all test govet gofmt dist
