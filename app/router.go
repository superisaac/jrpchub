package app

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jlib"
	"github.com/superisaac/jlib/http"
	"github.com/superisaac/rpcmap/mq"
	"sync"
	"time"
)

func NewRouter(ns string) *Router {
	return &Router{
		namespace:            ns,
		methodServicesIndex:  make(map[string][]*Service),
		methodRemoteServices: make(map[string][]*RemoteService),
	}
}

func (self *Router) App() *App {
	if self.app == nil {
		self.app = Application()
	}
	return self.app
}

func (self *Router) Start() {
	self.startOnce.Do(func() {
		go self.run(self.App().Context())
	})
}

func (self *Router) Stop() {
	if self.cancelFunc != nil {
		self.cancelFunc()
		self.cancelFunc = nil
		self.ctx = nil
		self.startOnce = sync.Once{}
	}
}

func (self *Router) Log() *log.Entry {
	return log.WithFields(log.Fields{
		"advurl":    self.App().Config.Server.AdvertiseUrl,
		"namespace": self.namespace,
	})
}

func (self *Router) Namespace() string {
	return self.namespace
}

func (self *Router) GetService(session jlibhttp.RPCSession) *Service {
	sid := session.SessionID()
	if v, ok := self.serviceIndex.Load(sid); ok {
		service, _ := v.(*Service)
		return service
	}
	service := NewService(self, session)
	self.serviceIndex.Store(sid, service)
	return service
}

func (self *Router) DismissService(sid string) bool {
	self.Log().Debugf("dismiss service %s", sid)
	if v, ok := self.serviceIndex.LoadAndDelete(sid); ok {
		service, _ := v.(*Service)
		// unlink methods
		service.UpdateMethods(nil)

		// send pending timeouts
		removing := []interface{}{}
		self.pendings.Range(func(k, v interface{}) bool {
			pt, _ := v.(*pendingT)
			if pt.toService == service {
				removing = append(removing, k)
			}
			return true
		})
		for _, k := range removing {
			if v, ok := self.pendings.LoadAndDelete(k); ok {
				// return a timeout messsage
				pt, _ := v.(*pendingT)
				timeout := jlib.ErrTimeout.ToMessage(pt.orig)
				pt.resultChannel <- timeout
			}
		}

		// dismiss the service
		service.Dismiss()
		return true
	} else {
		return false
	}
}

func (self *Router) checkExpire(reqId string, after time.Duration) {
	time.Sleep(after)
	if v, ok := self.pendings.LoadAndDelete(reqId); ok {
		pt, _ := v.(*pendingT)
		pt.orig.Log().Infof("request timeout ")
		pt.resultChannel <- jlib.ErrTimeout.ToMessage(pt.orig)
	}
}

func (self *Router) handleRequestMessage(reqmsg *jlib.RequestMessage) (interface{}, error) {
	if service, ok := self.SelectService(reqmsg.Method); ok {
		resultChannel := make(chan jlib.Message, 10)
		expireAfter := time.Second * 10
		pt := &pendingT{
			orig:          reqmsg,
			resultChannel: resultChannel,
			toService:     service,
			expiration:    time.Now().Add(expireAfter),
		}
		reqId := jlib.NewUuid()
		reqmsg = reqmsg.Clone(reqId)

		err := service.Send(reqmsg)
		if err != nil {
			return nil, err
		}
		self.pendings.Store(reqId, pt)
		go self.checkExpire(reqId, expireAfter)
		resmsg := <-resultChannel
		return resmsg, nil
	} else if rsrv, ok := self.SelectRemoteService(reqmsg.Method); ok {

		// select remote service
		c := rsrv.Client()
		resmsg, err := c.Call(context.Background(), reqmsg)
		return resmsg, err
	} else {
		return jlib.ErrMethodNotFound.ToMessage(reqmsg), nil
	}
}

func (self *Router) handleNotifyMessage(ntfmsg *jlib.NotifyMessage) (interface{}, error) {
	if service, ok := self.SelectService(ntfmsg.Method); ok {
		err := service.Send(ntfmsg)
		return nil, err
	} else {
		ntfmsg.Log().Debugf("delivered")
	}
	return nil, nil
}

func (self *Router) handleResultOrError(msg jlib.Message) (interface{}, error) {
	if v, ok := self.pendings.LoadAndDelete(msg.MustId()); ok {
		pt, _ := v.(*pendingT)

		if msg.IsResult() {
			resmsg := jlib.NewResultMessage(pt.orig, msg.MustResult())
			pt.resultChannel <- resmsg
		} else {
			// must be error
			errmsg := jlib.NewErrorMessage(pt.orig, msg.MustError())
			pt.resultChannel <- errmsg
		}
	} else {
		msg.Log().Warnf("cannot find pending requests")
	}
	return nil, nil
}

func (self *Router) Feed(msg jlib.Message) (interface{}, error) {
	if msg.IsRequest() {
		reqmsg, _ := msg.(*jlib.RequestMessage)
		return self.handleRequestMessage(reqmsg)
	} else if msg.IsNotify() {
		ntfmsg, _ := msg.(*jlib.NotifyMessage)
		return self.handleNotifyMessage(ntfmsg)
	} else {
		return self.handleResultOrError(msg)
	}
}

func (self *Router) run(rootctx context.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	self.ctx = ctx
	self.cancelFunc = cancel

	defer self.Stop()

	// TODO: listen channels
	self.Log().Debugf("router %s runs", self.namespace)

	statusSub := make(chan rpcmapmq.MQItem, 100)

	appcfg := self.App().Config
	if !appcfg.MQ.Empty() {
		self.mqClient = rpcmapmq.NewMQClient(appcfg.MQ.URL())
		go self.subscribeStatus(ctx, statusSub)
	}

	// publish the status
	err := self.publishStatus(ctx)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			err := self.publishEmptyStatus(context.Background())
			if err != nil {
				self.Log().Errorf("publish empty status error, %s", err)
			}
			return
		case <-time.After(time.Second * 15):
			// publish the status of
			err := self.publishStatus(ctx)
			if err != nil {
				panic(err)
			}
		case item, ok := <-statusSub:
			if !ok {
				return
			}
			self.updateStatus(item)
		}
	}
}

func (self *Router) updateStatus(item rpcmapmq.MQItem) {
	if item.Brief != "rpcmap.status" {
		return
	}
	ntf := item.Notify()
	var st serviceStatus

	err := jlib.DecodeInterface(ntf.Params[0], &st)
	if err != nil {
		self.Log().Errorf("bad decode service status: %s from notify %s", err, jlib.MessageString(ntf))
		return
	}

	if st.AdvertiseUrl == self.App().Config.Server.AdvertiseUrl {
		// self update
		return
	}

	if st.Timestamp < (time.Now().Add(-time.Minute * 2)).UTC().Unix() {
		// server status expired
		return
	}
	self.Log().Debugf("got service status advurl: %s, ts: %#v, methods: %+v", st.AdvertiseUrl, st.Timestamp, st.Methods)

	rsrv := self.GetOrCreateRemoteService(st.AdvertiseUrl)
	removed, added := rsrv.UpdateStatus(st)
	self.UpdateRemoteService(rsrv, removed, added)
}

func (self *Router) publishStatus(ctx context.Context) error {
	if self.mqClient == nil {
		return nil
	}
	if self.App().Config.Server.AdvertiseUrl == "" {
		self.Log().Warnf("advertise url is empty, server status will not be published, please add an advertise url in rpcmap.yml")
		return nil
	}

	methods := self.ServingMethods()
	status := serviceStatus{
		AdvertiseUrl: self.App().Config.Server.AdvertiseUrl,
		Methods:      methods,
		Timestamp:    time.Now().UTC().Unix(),
	}

	statusMap := map[string]interface{}{}
	err := jlib.DecodeInterface(status, &statusMap)
	if err != nil {
		return err
	}
	self.Log().Debugf("publish service status, %#v", statusMap)
	ntf := jlib.NewNotifyMessage("rpcmap.status", statusMap)
	section := "ns:" + self.namespace
	_, err = self.mqClient.Add(ctx, section, ntf)
	return err
}

func (self *Router) publishEmptyStatus(ctx context.Context) error {
	if self.mqClient == nil {
		return nil
	}
	if self.App().Config.Server.AdvertiseUrl == "" {
		self.Log().Warnf("advertise url is empty, server status will not be published, please add an advertise url in rpcmap.yml")
		return nil
	}

	status := serviceStatus{
		AdvertiseUrl: self.App().Config.Server.AdvertiseUrl,
		Methods:      []string{},
		Timestamp:    time.Now().UTC().Unix(),
	}

	statusMap := map[string]interface{}{}
	err := jlib.DecodeInterface(status, &statusMap)
	if err != nil {
		return err
	}
	self.Log().Debugf("publish empty service status, %#v", statusMap)
	ntf := jlib.NewNotifyMessage("rpcmap.status", statusMap)
	section := "ns:" + self.namespace
	_, err = self.mqClient.Add(ctx, section, ntf)
	return err
}

func (self *Router) subscribeStatus(rootctx context.Context, statusSub chan rpcmapmq.MQItem) {
	self.Log().Debugf("subscribe status")
	ctx, cancel := context.WithCancel(rootctx)
	defer cancel()

	section := "ns:" + self.namespace

	// prefetch some items
	chunk, err := self.mqClient.Tail(ctx, section, 10)
	if err != nil {
		self.Log().Errorf("tailing error %s", err)
	} else {
		for _, item := range chunk.Items {
			statusSub <- item
		}
	}

	if err := self.mqClient.Subscribe(ctx, section, statusSub); err != nil {
		self.Log().Errorf("subscribe error %s", err)
	}
}
