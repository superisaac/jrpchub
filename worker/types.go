package worker

import (
	"context"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
)

// client side structures
type WorkerRequest struct {
	Msg jsonz.Message
}

type WorkerCallback func(req *WorkerRequest, params []interface{}) (interface{}, error)

type WorkerHandler struct {
	callback WorkerCallback
	schema   jsonzschema.Schema
}

type WorkerHandlerSetter func(h *WorkerHandler)

type ServiceWorker struct {
	clients        []jsonzhttp.Streamable
	workerHandlers map[string]*WorkerHandler
	cancelFunc     func()
	connCtx        context.Context
}
