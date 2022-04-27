package worker

import (
	"context"
	//"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jlib"
	"github.com/superisaac/jlib/http"
	"sync"
)

func NewServiceWorker(serverUrls []string) *ServiceWorker {
	actor := jlibhttp.NewActor()
	worker := &ServiceWorker{
		Actor:   actor,
		clients: []jlibhttp.Streamable{},
	}

	for _, serverUrl := range serverUrls {
		client := worker.initClient(serverUrl)
		worker.clients = append(worker.clients, client)
	}
	actor.On("_ping", func(params []interface{}) (interface{}, error) {
		return "pong", nil
	})
	return worker
}

func (self *ServiceWorker) initClient(serverUrl string) jlibhttp.Streamable {
	client, err := jlibhttp.NewClient(serverUrl)
	if err != nil {
		log.Panicf("new client %s", err)
	}
	sc, ok := client.(jlibhttp.Streamable)
	if !ok {
		log.Panicf("client is not streamable")
	}
	sc.OnMessage(func(msg jlib.Message) {
		err := self.feed(msg, sc)
		if err != nil {
			msg.Log().Errorf("feed error %s", err)
		}
		// if msg.IsRequest() {
		// 	reqmsg, _ := msg.(*jlib.RequestMessage)
		// 	self.feedRequest(reqmsg, sc)
		// } else if msg.IsNotify() {
		// 	ntfmsg, _ := msg.(*jlib.NotifyMessage)
		// 	self.feedNotify(ntfmsg, sc)
		// } else {
		// 	msg.Log().Info("worker got message")
		// }
	})
	return sc
}

func (self *ServiceWorker) feed(msg jlib.Message, client jlibhttp.Streamable) error {
	req := jlibhttp.NewRPCRequest(self.connCtx, msg, jlibhttp.TransportHTTP, nil)

	resmsg, err := self.Actor.Feed(req)
	if err != nil {
		return err
	}
	if resmsg != nil {
		client.Send(self.connCtx, resmsg)
	}
	return nil
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
		go func(idx int, c jlibhttp.Streamable) {
			err := self.connectClient(ctx, wg, c)
			if err != nil {
				log.Errorf("error connect client %d: %s", idx, err)
			}
		}(i, client)
	}
	wg.Wait()
}

func (self *ServiceWorker) connectClient(rootCtx context.Context, wg *sync.WaitGroup, client jlibhttp.Streamable) error {
	defer wg.Done()

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		return err
	}

	// declare methods
	methods := map[string]interface{}{}
	for _, mname := range self.Actor.MethodList() {
		if !jlib.IsPublicMethod(mname) {
			continue
		}
		if s, ok := self.Actor.GetSchema(mname); ok {
			methods[mname] = s.Map()
		} else {
			methods[mname] = nil
		}
	}
	reqmsg := jlib.NewRequestMessage(jlib.NewUuid(), "rpc.declare", []interface{}{methods})
	resmsg, err := client.Call(ctx, reqmsg)
	if err != nil {
		return err
	} else if resmsg.IsError() {
		return resmsg.MustError()
	}
	// resmsg should be "ok"

	return client.Wait()
}
