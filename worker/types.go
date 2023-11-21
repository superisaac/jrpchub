package worker

import (
	"context"
	"github.com/superisaac/jsoff/net"
)

// client side structures
type ServiceWorker struct {
	Actor      *jsoffnet.Actor
	clients    []jsoffnet.Streamable
	cancelFunc func()
	connCtx    context.Context
}
