package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcmap/app"
	"net/http"
	"os"
	"time"
)

func setupLogger(logOutput string) {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	if logOutput == "" {
		logOutput = os.Getenv("LOG_OUTPUT")
	}
	if logOutput == "" || logOutput == "console" || logOutput == "stdout" {
		log.SetOutput(os.Stdout)
	} else if logOutput == "stderr" {
		log.SetOutput(os.Stderr)
	} else {
		file, err := os.OpenFile(logOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		log.SetOutput(file)
	}

	envLogLevel := os.Getenv("LOG_LEVEL")
	switch envLogLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func StartServer() {
	flagset := flag.NewFlagSet("rpcmap", flag.ExitOnError)

	// bind address
	pBind := flagset.String("bind", "", "bind address, default is 127.0.0.1:6000")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	pYamlConfig := flagset.String("config", "", "path to <server config.yml>")

	// parse command-line flags
	flagset.Parse(os.Args[1:])
	setupLogger(*pLogfile)

	application := app.GetApp()

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
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jsonzhttp.NewAuthHandler(application.Config.Server.Auth, handler)
	log.Infof("rpcmap starts at %s with secureness %t", bind, !insecure)
	jsonzhttp.ListenAndServe(rootCtx, bind, handler, application.Config.Server.TLS)
}

func main() {
	StartServer()
}
