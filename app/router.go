package rpczapp

import (
	//"fmt"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"math/rand"
	//log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	routerOnce sync.Once
)

var router *Router

func GetRouter() *Router {
	routerOnce.Do(func() {
		router = NewRouter()
	})
	return router
}

func NewRouter() *Router {
	return &Router{
		serviceIndex:        make(map[string]*Service),
		methodServicesIndex: make(map[string][]ServiceInfo),
	}
}

func (self *Router) GetService(session jsonzhttp.RPCSession) *Service {
	sid := session.SessionID()
	if service, ok := self.serviceIndex[sid]; ok {
		return service
	}
	service := NewService(self, session)
	self.serviceIndex[sid] = service
	return service
}

func (self *Router) RemoveService(sid string) {
	if service, ok := self.serviceIndex[sid]; ok {
		delete(self.serviceIndex, sid)
		service.OnRemoved()
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
			expiration:    time.Now().Add(expireAfter),
		}
		reqId := jsonz.NewUuid()
		reqmsg = reqmsg.Clone(reqId)
		self.pendings.Store(reqId, pt)
		err := service.Send(reqmsg)
		if err != nil {
			return nil, err
		}
		go self.checkExpire(reqId, expireAfter)
	} else {
		return jsonz.ErrMethodNotFound.ToMessage(reqmsg), nil
	}
	return nil, nil
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
