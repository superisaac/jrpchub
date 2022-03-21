package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcz/app"
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
	pBind := flagset.String("bind", "127.0.0.1:6000", "bind address")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	// cert flags
	pCertfile := flagset.String("cert", "", "path to cert file")
	pKeyfile := flagset.String("key", "", "path to key file")

	// parse command-line flags
	flagset.Parse(os.Args[1:])

	rootCtx := context.Background()

	// insecure mode
	insecure := true
	var tlscfg *jsonzhttp.TLSConfig

	if *pCertfile != "" && *pKeyfile != "" {
		// secure mode
		insecure = false
		tlscfg = &jsonzhttp.TLSConfig{
			Certfile: *pCertfile,
			Keyfile:  *pKeyfile,
		}
	}

	setupLogger(*pLogfile)

	actor := rpczapp.NewActor()
	var handler http.Handler
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jsonzhttp.NewAuthHandler(nil, handler)
	log.Infof("rpcz starts at %s with secureness %t", *pBind, !insecure)
	jsonzhttp.ListenAndServe(rootCtx, *pBind, handler, tlscfg)
}

func main() {
	StartServer()
}
