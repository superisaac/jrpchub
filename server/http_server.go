package main

import (
	"context"
	"flag"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcz/app"
	"net/http"
)

func StartServer() {
	flagset := flag.NewFlagSet("jsonz-example-fifo", flag.ExitOnError)
	pBind := flagset.String("bind", "127.0.0.1:6000", "bind address")

	rootCtx := context.Background()
	insecure := true

	actor := rpczapp.NewActor()
	var handler http.Handler
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jsonzhttp.NewAuthHandler(nil, handler)
	jsonzhttp.ListenAndServe(rootCtx, *pBind, handler)
}

func main() {
	StartServer()
}
