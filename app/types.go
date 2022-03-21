package rpczapp

import (
	"context"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"sync"
	"time"
)

type ServiceInfo struct {
	service *Service
	// TODO: other fields, such as weights
}

type pendingT struct {
	orig          *jsonz.RequestMessage
	resultChannel chan jsonz.Message
	expiration    time.Time
}

type Router struct {
	serviceIndex        map[string]*Service
	methodServicesIndex map[string][]ServiceInfo
	pendings            sync.Map
}

type Service struct {
	router  *Router
	session jsonzhttp.RPCSession
	methods map[string]jsonzschema.Schema
}

// client side structures
type WorkerRequest struct {
	Msg jsonz.Message
}

type WorkerCallback func(req *WorkerRequest, params []interface{}) (interface{}, error)

type ServiceWorker struct {
	connCtx         context.Context
	client          jsonzhttp.Streamable
	workerCallbacks map[string]WorkerCallback
}
