package main

import (
	"context"
	"flag"
	//"fmt"
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
	flagset := flag.NewFlagSet("jsonz-example-fifo", flag.ExitOnError)

	// bind address
	pBind := flagset.String("bind", "", "bind address, default is 127.0.0.1:6000")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	pYamlConfig := flagset.String("config", "", "path to <server config.yml>")

	// parse command-line flags
	flagset.Parse(os.Args[1:])
	setupLogger(*pLogfile)

	app := rpcmapapp.GetApp()

	if *pYamlConfig != "" {
		err := app.Config.Load(*pYamlConfig)
		if err != nil {
			log.Panicf("load config error %s", err)
		}
	}

	bind := *pBind
	if bind == "" {
		bind = app.Config.Server.Bind
	}

	if bind == "" {
		bind = "127.0.0.1:6000"
	}

	insecure := app.Config.Server.TLS == nil

	//rpcmapCfg := serverConfig.(RPCMAPConfig)
	rootCtx := context.Background()

	// start default router
	_ = app.GetRouter("default")
	actor := rpcmapapp.NewActor()
	var handler http.Handler
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jsonzhttp.NewAuthHandler(app.Config.Server.Auth, handler)
	log.Infof("rpcmap starts at %s with secureness %t", bind, !insecure)
	jsonzhttp.ListenAndServe(rootCtx, bind, handler, app.Config.Server.TLS)
}

func main() {
	StartServer()
}
