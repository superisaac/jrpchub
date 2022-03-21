package rpczapp

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
)

func NewServiceWorker(serverUrl string) *ServiceWorker {
	client, err := jsonzhttp.NewClient(serverUrl)
	if err != nil {
		log.Panicf("new client %s", err)
	}
	sc, ok := client.(jsonzhttp.Streamable)
	if !ok {
		log.Panicf("client is not streamable")
	}

	worker := &ServiceWorker{
		client: sc,
	}
	sc.OnMessage(func(msg jsonz.Message) {
		if msg.IsRequest() {
			reqmsg, _ := msg.(*jsonz.RequestMessage)
			worker.feedRequest(reqmsg)
		} else if msg.IsNotify() {
			ntfmsg, _ := msg.(*jsonz.NotifyMessage)
			worker.feedNotify(ntfmsg)
		} else {
			msg.Log().Info("worker got message")
		}
	})
	return worker
}

func (self *ServiceWorker) On(method string, callback WorkerCallback) error {
	if _, ok := self.workerCallbacks[method]; ok {
		return errors.New("callback already exist")
	}
	self.workerCallbacks[method] = callback
	return nil
}

func (self *ServiceWorker) feedRequest(reqmsg *jsonz.RequestMessage) {
	if cb, ok := self.workerCallbacks[reqmsg.Method]; ok {
		req := &WorkerRequest{
			Msg: reqmsg,
		}
		res, err := cb(req, reqmsg.Params)
		resmsg, err := self.wrapResult(res, err, reqmsg)
		self.client.Send(self.connCtx, resmsg)
	} else {
		errmsg := jsonz.ErrMethodNotFound.ToMessage(reqmsg)
		self.client.Send(self.connCtx, errmsg)
	}
}

func (self *ServiceWorker) feedNotify(ntfmsg *jsonz.NotifyMessage) {
	if cb, ok := self.workerCallbacks[ntfmsg.Method]; ok {
		req := &WorkerRequest{
			Msg: ntfmsg,
		}
		res, err := cb(req, ntfmsg.Params)
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

func (self *ServiceWorker) ConnectWait(rootCtx context.Context) error {
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	err := self.client.Connect(ctx)
	if err != nil {
		return err
	}
	self.connCtx = ctx

	defer func() {
		self.connCtx = nil
	}()

	// declare methods
	methods := []string{}
	for mname, _ := range self.workerCallbacks {
		methods = append(methods, mname)
	}
	reqmsg := jsonz.NewRequestMessage(jsonz.NewUuid(), "rpcz.declare", []interface{}{methods})
	resmsg, err := self.client.Call(ctx, reqmsg)
	if err != nil {
		return err
	} else if resmsg.IsError() {
		return resmsg.MustError()
	}
	// resmsg should be "ok"

	return self.client.Wait()
}
