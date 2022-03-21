package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcz/app"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type ServerConfig struct {
	Bind string                `yaml:"bind"`
	Auth *jsonzhttp.AuthConfig `yaml:"auth,omitempty"`
	TLS  *jsonzhttp.TLSConfig  `yaml:"tls,omitempty"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{}
}

func (self *ServerConfig) Load(yamlPath string) error {
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return nil
	}

	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return err
	}
	return self.LoadYamldata(data)
}

func (self *ServerConfig) LoadYamldata(yamlData []byte) error {
	err := yaml.Unmarshal(yamlData, self)
	if err != nil {
		return err
	}
	return self.validateValues()
}

func (self *ServerConfig) validateValues() error {
	if self.TLS != nil {
		err := self.TLS.ValidateValues()
		if err != nil {
			return err
		}
	}
	if self.Auth != nil {
		err := self.Auth.ValidateValues()
		if err != nil {
			return err
		}
	}
	return nil
}

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

	serverConfig := NewServerConfig()
	if *pYamlConfig != "" {
		err := serverConfig.Load(*pYamlConfig)
		if err != nil {
			log.Panicf("load config error %s", err)
		}
	}

	bind := *pBind
	if bind == "" {
		bind = serverConfig.Bind
	}

	if bind == "" {
		bind = "127.0.0.1:6000"
	}

	insecure := serverConfig.TLS == nil

	rootCtx := context.Background()
	actor := rpczapp.NewActor()
	var handler http.Handler
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, insecure)
	handler = jsonzhttp.NewAuthHandler(serverConfig.Auth, handler)
	log.Infof("rpcz starts at %s with secureness %t", bind, !insecure)
	jsonzhttp.ListenAndServe(rootCtx, bind, handler, serverConfig.TLS)
}

func main() {
	StartServer()
}
