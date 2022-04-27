package worker

import (
	"context"
	"github.com/superisaac/jlib/http"
)

// client side structures
type ServiceWorker struct {
	Actor      *jlibhttp.Actor
	clients    []jlibhttp.Streamable
	cancelFunc func()
	connCtx    context.Context
}
