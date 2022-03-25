package rpcmapworker

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"sync"
)

func WithSchema(s jsonzschema.Schema) WorkerHandlerSetter {
	return func(h *WorkerHandler) {
		h.schema = s
	}
}

func WithSchemaYaml(yamlSchema string) WorkerHandlerSetter {
	builder := jsonzschema.NewSchemaBuilder()
	s, err := builder.BuildYamlBytes([]byte(yamlSchema))
	if err != nil {
		panic(err)
	}
	return WithSchema(s)
}

func WithSchemaJson(jsonSchema string) WorkerHandlerSetter {
	builder := jsonzschema.NewSchemaBuilder()
	s, err := builder.BuildBytes([]byte(jsonSchema))
	if err != nil {
		panic(err)
	}
	return WithSchema(s)
}

func NewServiceWorker(serverUrls []string) *ServiceWorker {
	worker := &ServiceWorker{
		clients:        []jsonzhttp.Streamable{},
		workerHandlers: make(map[string]*WorkerHandler),
	}
	for _, serverUrl := range serverUrls {
		client := worker.initClient(serverUrl)
		worker.clients = append(worker.clients, client)
	}
	return worker
}

func (self *ServiceWorker) initClient(serverUrl string) jsonzhttp.Streamable {
	client, err := jsonzhttp.NewClient(serverUrl)
	if err != nil {
		log.Panicf("new client %s", err)
	}
	sc, ok := client.(jsonzhttp.Streamable)
	if !ok {
		log.Panicf("client is not streamable")
	}
	sc.OnMessage(func(msg jsonz.Message) {
		if msg.IsRequest() {
			reqmsg, _ := msg.(*jsonz.RequestMessage)
			self.feedRequest(reqmsg, sc)
		} else if msg.IsNotify() {
			ntfmsg, _ := msg.(*jsonz.NotifyMessage)
			self.feedNotify(ntfmsg, sc)
		} else {
			msg.Log().Info("worker got message")
		}
	})
	return sc
}

func (self *ServiceWorker) On(method string, callback WorkerCallback, setters ...WorkerHandlerSetter) error {
	if _, ok := self.workerHandlers[method]; ok {
		return errors.New("callback already exist")
	}

	h := &WorkerHandler{
		callback: callback,
	}

	for _, setter := range setters {
		setter(h)
	}
	self.workerHandlers[method] = h
	return nil
}

// register a typed method handler
func (self *ServiceWorker) OnTyped(method string, typedHandler interface{}, setters ...WorkerHandlerSetter) error {
	firstArg := &WorkerRequest{}
	handler, err := wrapTyped(typedHandler, firstArg)
	if err != nil {
		return err
	}
	return self.On(method, handler, setters...)
}

func (self *ServiceWorker) feedRequest(reqmsg *jsonz.RequestMessage, client jsonzhttp.Streamable) {
	if h, ok := self.workerHandlers[reqmsg.Method]; ok {
		req := &WorkerRequest{
			Msg: reqmsg,
		}
		res, err := h.callback(req, reqmsg.Params)
		resmsg, err := self.wrapResult(res, err, reqmsg)
		client.Send(self.connCtx, resmsg)
	} else {
		errmsg := jsonz.ErrMethodNotFound.ToMessage(reqmsg)
		client.Send(self.connCtx, errmsg)
	}
}

func (self *ServiceWorker) feedNotify(ntfmsg *jsonz.NotifyMessage, client jsonzhttp.Streamable) {
	if h, ok := self.workerHandlers[ntfmsg.Method]; ok {
		req := &WorkerRequest{
			Msg: ntfmsg,
		}
		res, err := h.callback(req, ntfmsg.Params)
		if err != nil {
			ntfmsg.Log().Errorf("notify error %s", err)
		} else if res != nil {
			// discard res
			ntfmsg.Log().Debugf("notify result %+v", res)
		}
	}
}

func (self ServiceWorker) wrapResult(res interface{}, err error, reqmsg *jsonz.RequestMessage) (jsonz.Message, error) {
	if err != nil {
		var rpcErr *jsonz.RPCError
		if errors.As(err, &rpcErr) {
			return rpcErr.ToMessage(reqmsg), nil
		} else {
			return jsonz.ErrInternalError.ToMessage(reqmsg), nil
		}
	} else if resmsg1, ok := res.(jsonz.Message); ok {
		// normal response
		return resmsg1, nil
	} else {
		return jsonz.NewResultMessage(reqmsg, res), nil
	}
}

func (self *ServiceWorker) ConnectWait(rootCtx context.Context) {
	if self.cancelFunc != nil {
		log.Warnf("worker already connected")
		return
	}
	ctx, cancel := context.WithCancel(rootCtx)
	self.cancelFunc = cancel
	self.connCtx = ctx

	defer func() {
		self.cancelFunc()
		self.cancelFunc = nil
		self.connCtx = nil
	}()

	wg := &sync.WaitGroup{}

	for i, client := range self.clients {
		wg.Add(1)
		go func(idx int, c jsonzhttp.Streamable) {
			err := self.connectClient(ctx, wg, c)
			if err != nil {
				log.Errorf("error connect client %d: %s", idx, err)
			}
		}(i, client)
	}
	wg.Wait()
}

func (self *ServiceWorker) connectClient(rootCtx context.Context, wg *sync.WaitGroup, client jsonzhttp.Streamable) error {
	defer wg.Done()

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		return err
	}

	// declare methods
	methods := map[string]interface{}{}
	for mname, h := range self.workerHandlers {
		//methods = append(methods, mname)
		if h.schema != nil {
			methods[mname] = h.schema.RebuildType()
		} else {
			methods[mname] = nil
		}
	}
	reqmsg := jsonz.NewRequestMessage(jsonz.NewUuid(), "rpc.declare", []interface{}{methods})
	resmsg, err := client.Call(ctx, reqmsg)
	if err != nil {
		return err
	} else if resmsg.IsError() {
		return resmsg.MustError()
	}
	// resmsg should be "ok"

	return client.Wait()
}
