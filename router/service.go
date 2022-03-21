package rpczrouter

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

func (self *Service) UpdateMethods(newMethods map[string]jsonzschema.Schema) {
	if self.router == nil {
		log.Errorf("cannot update methods on removed service")
		return
	}
	removed := make([]string, 0)
	added := make([]string, 0)

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
}

func (self *Service) OnRemoved() {
	self.router = nil
	self.session = nil
}

func (self *Service) Send(msg jsonz.Message) error {
	// TODO: schema test
	self.session.Send(msg)
	return nil
}
