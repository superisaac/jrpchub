package rpcmapapp

import (
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"sync"
	"time"
)

type RPCMAPConfig struct {
	MQUrl string `yaml:"mqurl,omitempty"`
}

type ServiceInfo struct {
	service *Service
	// TODO: other fields, such as weights
}

type pendingT struct {
	orig          *jsonz.RequestMessage
	resultChannel chan jsonz.Message
	toService     *Service
	expiration    time.Time
}

type Router struct {
	startOnce           sync.Once
	serviceIndex        sync.Map
	methodServicesIndex map[string][]ServiceInfo
	pendings            sync.Map
}

type Service struct {
	router  *Router
	session jsonzhttp.RPCSession
	methods map[string]jsonzschema.Schema
}
