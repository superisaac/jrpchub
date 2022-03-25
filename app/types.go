package rpcmapapp

import (
	"context"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"github.com/superisaac/rpcmap/mq"
	"sync"
	"time"
)

type AppConfig struct {
	Server struct {
		Bind         string                `yaml:"bind"`
		AdvertiseUrl string                `yaml:"advertise_url,omitempty"`
		Auth         *jsonzhttp.AuthConfig `yaml:"auth,omitempty"`
		TLS          *jsonzhttp.TLSConfig  `yaml:"tls,omitempty"`
	} `yaml:"server"`

	MQ struct {
		Url string `yaml:"url"`
	} `yaml:"mq,omitempty"`
}

type pendingT struct {
	orig          *jsonz.RequestMessage
	resultChannel chan jsonz.Message
	toService     *Service
	expiration    time.Time
}

type serviceStatus struct {
	AdvertiseUrl string    `json:"advertise_url"`
	Methods      []string  `json:"methods"`
	Timestamp    time.Time `json:"timestamp"`
}

type RemoteService struct {
	AdvertiseUrl string
	Methods      map[string]bool
	UpdateAt     time.Time

	client jsonzhttp.Client
}

type Router struct {
	namespace string

	// context
	ctx        context.Context
	cancelFunc func()

	// start
	startOnce sync.Once

	// service indices
	serviceIndex        sync.Map
	methodServicesIndex map[string][]*Service

	// remote service indices
	remoteServiceIndex   sync.Map
	methodRemoteServices map[string][]*RemoteService

	// pending requests
	pendings sync.Map

	// mq
	mqClient rpcmapmq.MQClient
}

type Service struct {
	router  *Router
	session jsonzhttp.RPCSession
	methods map[string]jsonzschema.Schema
}
