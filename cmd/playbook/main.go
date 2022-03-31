package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/rpcmap/playbook"
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

func StartPlaybook() {
	flagset := flag.NewFlagSet("rpcmap-playbook", flag.ExitOnError)

	// connect to server
	pConnect := flagset.String("c", "h2c://127.0.0.1:6000", "connect to rpcmap server")

	// logging flags
	pYaml := flagset.String("config", "playbook.yml", "path to playbook.yml")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	// parse command-line flags
	flagset.Parse(os.Args[1:])
	setupLogger(*pLogfile)

	pb := playbook.NewPlaybook()
	if err := pb.Config.Load(*pYaml); err != nil {
		panic(err)
	}

	if err := pb.Run(context.Background(), *pConnect); err != nil {
		panic(err)
	}
}

func main() {
	StartPlaybook()
}
