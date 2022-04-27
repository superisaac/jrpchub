package worker

import (
	"context"
	"github.com/superisaac/jlib"
	"github.com/superisaac/jlib/http"
	"github.com/superisaac/jlib/schema"
)

// client side structures
type WorkerRequest struct {
	Msg jlib.Message
}

type WorkerCallback func(req *WorkerRequest, params []interface{}) (interface{}, error)
type WorkerMsgCallback func(params []interface{}) (interface{}, error)

type WorkerHandler struct {
	callback WorkerCallback
	schema   jlibschema.Schema
}

type WorkerHandlerSetter func(h *WorkerHandler)

type ServiceWorker struct {
	clients        []jlibhttp.Streamable
	workerHandlers map[string]*WorkerHandler
	cancelFunc     func()
	connCtx        context.Context
}
