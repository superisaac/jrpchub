package rpczapp

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
)

func NewService(router *Router, session jsonzhttp.RPCSession) *Service {
	return &Service{
		router:  router,
		session: session,
		methods: make(map[string]jsonzschema.Schema),
	}
}

func (self *Service) UpdateMethods(newMethods map[string]jsonzschema.Schema) error {
	if self.router == nil {
		log.Errorf("cannot update methods on removed service")
		return jsonz.ParamsError("update methods failed")
	}
	if newMethods == nil {
		// clean methods
		newMethods = map[string]jsonzschema.Schema{}
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

func (self *Service) Send(msg jsonz.Message) error {
	// TODO: schema test
	self.session.Send(msg)
	return nil
}

func (self *Service) GetSchema(method string) (jsonzschema.Schema, bool) {
	if s, ok := self.methods[method]; ok && s != nil {
		return s, true
	}
	return nil, false
}
