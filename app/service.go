package app

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jlib"
	"github.com/superisaac/jlib/http"
	"github.com/superisaac/jlib/schema"
	"math/rand"
)

func NewService(router *Router, session jlibhttp.RPCSession) *Service {
	return &Service{
		router:  router,
		session: session,
		methods: make(map[string]jlibschema.Schema),
	}
}

func (self *Service) UpdateMethods(newMethods map[string]jlibschema.Schema) error {
	if self.router == nil {
		log.Errorf("cannot update methods on removed service")
		return jlib.ParamsError("update methods failed")
	}
	if newMethods == nil {
		// clean methods
		newMethods = map[string]jlibschema.Schema{}
	}
	removed := []string{}
	added := []string{}

	for mname, _ := range self.methods {
		if _, ok := newMethods[mname]; !ok {
			// not present in new methods
			removed = append(removed, mname)
		}
	}

	for mname, _ := range newMethods {
		if _, ok := self.methods[mname]; !ok {
			// not present in self.methods
			added = append(added, mname)
		}
	}

	self.methods = newMethods
	self.router.UpdateService(self, removed, added)
	return nil
}

func (self *Service) Dismiss() {
	self.router = nil
	self.session = nil
}

func (self *Service) Send(msg jlib.Message) error {
	// TODO: schema test
	self.session.Send(msg)
	return nil
}

func (self *Service) GetSchema(method string) (jlibschema.Schema, bool) {
	if s, ok := self.methods[method]; ok && s != nil {
		return s, true
	}
	return nil, false
}

// router methods related to services
// services methods
func (self *Router) AddService(method string, service *Service) {
	self.serviceLock.Lock()
	defer self.serviceLock.Unlock()

	srvs, ok := self.methodServicesIndex[method]
	if !ok {
		srvs = make([]*Service, 0)
	}
	self.methodServicesIndex[method] = append(srvs, service)
}

func (self *Router) RemoveService(method string, service *Service) (changed bool) {
	self.serviceLock.Lock()
	defer self.serviceLock.Unlock()

	if srvs, ok := self.methodServicesIndex[method]; ok {
		found := -1
		for i, srv := range srvs {
			if srv == service {
				found = i
				break
			}
		}
		if found >= 0 {
			if len(srvs) <= 1 {
				if found != 0 {
					log.Panicf("services list has only 1 elements but found %d is not 0, method=%s", found, method)
				}
				// if only one service element, just remove it
				delete(self.methodServicesIndex, method)
			} else {
				// removed srvs[found]
				self.methodServicesIndex[method] = append(srvs[:found], srvs[found+1:]...)
			}
			return true
		}
	}
	return false
}

func (self *Router) UpdateService(service *Service, removed []string, added []string) {
	changed := false
	for _, mname := range removed {
		self.RemoveService(mname, service)
		changed = true
	}
	for _, mname := range added {
		self.AddService(mname, service)
		changed = true
	}

	if changed && self.ctx != nil {
		go self.publishStatus(self.ctx)
	}
}

func (self *Router) SelectService(method string) (*Service, bool) {
	self.serviceLock.RLock()
	defer self.serviceLock.RUnlock()

	if srvs, ok := self.methodServicesIndex[method]; ok && len(srvs) > 0 {
		idx := rand.Intn(len(srvs))
		return srvs[idx], true
	}
	return nil, false
}

func (self *Router) ServingMethods() []string {
	self.serviceLock.RLock()
	defer self.serviceLock.RUnlock()

	methods := []string{}
	for mname, _ := range self.methodServicesIndex {
		methods = append(methods, mname)
	}
	return methods
}
