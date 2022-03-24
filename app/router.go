package rpczapp

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"math/rand"
	"sync"
	"time"
)

var (
	routers sync.Map
)

func GetRouter(ns string) *Router {
	if v, ok := routers.Load(ns); ok {
		router, _ := v.(*Router)
		return router
	} else {
		v, loaded := routers.LoadOrStore(ns, NewRouter())
		if loaded {
			log.Warnf("routers concurrent load")
		}
		router, _ := v.(*Router)
		router.Start()
		return router
	}
}

func NewRouter() *Router {
	return &Router{
		methodServicesIndex: make(map[string][]ServiceInfo),
	}
}

func (self *Router) Start() {
	self.startOnce.Do(func() {
		go self.Run()
	})
}

func (self *Router) GetService(session jsonzhttp.RPCSession) *Service {
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
				timeout := jsonz.ErrTimeout.ToMessage(pt.orig)
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

func (self *Router) Add(method string, service *Service) {
	infos, ok := self.methodServicesIndex[method]
	if !ok {
		infos = make([]ServiceInfo, 0)
	}
	info := ServiceInfo{service: service}
	self.methodServicesIndex[method] = append(infos, info)
}

func (self *Router) Remove(method string, service *Service) bool {
	if infos, ok := self.methodServicesIndex[method]; ok {
		found := -1
		for i, info := range infos {
			if info.service == service {
				found = i
				break
			}
		}
		if found >= 0 {
			// removed infos[found]
			self.methodServicesIndex[method] = append(infos[:found], infos[found+1:]...)
			return true
		}

	}
	return false
}

func (self *Router) UpdateService(service *Service, removed []string, added []string) {
	for _, mname := range removed {
		self.Remove(mname, service)
	}
	for _, mname := range added {
		self.Add(mname, service)
	}
}

func (self *Router) SelectService(method string) (*Service, bool) {
	if infos, ok := self.methodServicesIndex[method]; ok && len(infos) > 0 {
		idx := rand.Intn(len(infos))
		return infos[idx].service, true
	}
	return nil, false
}

func (self *Router) checkExpire(reqId string, after time.Duration) {
	time.Sleep(after)
	if v, ok := self.pendings.LoadAndDelete(reqId); ok {
		pt, _ := v.(*pendingT)
		pt.orig.Log().Infof("request timeout ")
		pt.resultChannel <- jsonz.ErrTimeout.ToMessage(pt.orig)
	}
}

func (self *Router) handleRequestMessage(reqmsg *jsonz.RequestMessage) (interface{}, error) {
	if service, ok := self.SelectService(reqmsg.Method); ok {
		resultChannel := make(chan jsonz.Message, 10)
		expireAfter := time.Second * 10
		pt := &pendingT{
			orig:          reqmsg,
			resultChannel: resultChannel,
			toService:     service,
			expiration:    time.Now().Add(expireAfter),
		}
		reqId := jsonz.NewUuid()
		reqmsg = reqmsg.Clone(reqId)

		err := service.Send(reqmsg)
		if err != nil {
			return nil, err
		}
		self.pendings.Store(reqId, pt)
		go self.checkExpire(reqId, expireAfter)
		resmsg := <-resultChannel
		return resmsg, nil
	} else {
		return jsonz.ErrMethodNotFound.ToMessage(reqmsg), nil
	}
}

func (self *Router) handleNotifyMessage(ntfmsg *jsonz.NotifyMessage) (interface{}, error) {
	if service, ok := self.SelectService(ntfmsg.Method); ok {
		err := service.Send(ntfmsg)
		return nil, err
	} else {
		ntfmsg.Log().Infof("delivered")
	}
	return nil, nil
}

func (self *Router) handleResultOrError(msg jsonz.Message) (interface{}, error) {
	if v, ok := self.pendings.LoadAndDelete(msg.MustId()); ok {
		pt, _ := v.(*pendingT)

		if msg.IsResult() {
			resmsg := jsonz.NewResultMessage(pt.orig, msg.MustResult())
			pt.resultChannel <- resmsg
		} else {
			// must be error
			errmsg := jsonz.NewErrorMessage(pt.orig, msg.MustError())
			pt.resultChannel <- errmsg
		}
	} else {
		msg.Log().Warnf("cannot find pending requests")
	}
	return nil, nil
}

func (self *Router) Feed(msg jsonz.Message) (interface{}, error) {
	if msg.IsRequest() {
		reqmsg, _ := msg.(*jsonz.RequestMessage)
		return self.handleRequestMessage(reqmsg)
	} else if msg.IsNotify() {
		ntfmsg, _ := msg.(*jsonz.NotifyMessage)
		return self.handleNotifyMessage(ntfmsg)
	} else {
		return self.handleResultOrError(msg)
	}
}

func (self *Router) Run() {
	// TODO: listen channels
}
