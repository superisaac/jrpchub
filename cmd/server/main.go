package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jlib/http"
	"github.com/superisaac/rpcmap/app"
	"github.com/superisaac/rpcmap/cmd/cmdutil"
	"net/http"
	"os"
)

func StartServer() {
	flagset := flag.NewFlagSet("rpcmap", flag.ExitOnError)

	// bind address
	pBind := flagset.String("bind", "", "bind address, default is 127.0.0.1:6000")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	pYamlConfig := flagset.String("config", "", "path to <server config.yml>")

	// parse command-line flags
	flagset.Parse(os.Args[1:])
	cmdutil.SetupLogger(*pLogfile)

	application := app.Application()

	if *pYamlConfig != "" {
		err := application.Config.Load(*pYamlConfig)
		if err != nil {
			log.Panicf("load config error %s", err)
		}
	}

	bind := *pBind
	if bind == "" {
		bind = application.Config.Server.Bind
	}

	if bind == "" {
		bind = "127.0.0.1:6000"
	}

	insecure := application.Config.Server.TLS == nil

	//rpcmapCfg := serverConfig.(RPCMAPConfig)
	rootCtx := context.Background()

	// start default router
	_ = application.GetRouter("default")
	actor := app.NewActor()
	var handler http.Handler
	handler = jlibhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jlibhttp.NewAuthHandler(application.Config.Server.Auth, handler)
	log.Infof("rpcmap starts at %s with secureness %t", bind, !insecure)
	jlibhttp.ListenAndServe(rootCtx, bind, handler, application.Config.Server.TLS)
}

func main() {
	StartServer()
}
