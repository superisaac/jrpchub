package app

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz/http"
	"math/rand"
	"time"
)

func (self *RemoteService) Client() jsonzhttp.Client {
	if self.client == nil {
		c, err := jsonzhttp.NewClient(self.AdvertiseUrl)
		if err != nil {
			log.Panicf("error create remote client: %s", err)
		}
		self.client = c
	}
	return self.client
}

func (self *RemoteService) UpdateStatus(newStatus serviceStatus) ([]string, []string) {
	newMethods := map[string]bool{}
	for _, mname := range newStatus.Methods {
		newMethods[mname] = true
	}

	removed := []string{}
	added := []string{}

	if self.Methods == nil {
		self.Methods = map[string]bool{}
	}

	for mname, _ := range self.Methods {
		if _, ok := newMethods[mname]; !ok {
			removed = append(removed, mname)
		}
	}

	for mname, _ := range newMethods {
		if _, ok := self.Methods[mname]; !ok {
			// not present in self.Methods
			added = append(added, mname)
		}
	}
	self.Methods = newMethods
	self.AdvertiseUrl = newStatus.AdvertiseUrl
	self.UpdateAt = time.Unix(newStatus.Timestamp, 0)
	return removed, added
}

// remote services methods
func (self *Router) GetOrCreateRemoteService(advUrl string) *RemoteService {
	if v, ok := self.remoteServiceIndex.Load(advUrl); ok {
		rsrv, _ := v.(*RemoteService)
		return rsrv
	} else {
		newsrv := &RemoteService{AdvertiseUrl: advUrl}
		v, _ := self.remoteServiceIndex.LoadOrStore(advUrl, newsrv)
		rsrv, _ := v.(*RemoteService)
		return rsrv
	}
}

func (self *Router) AddRemote(method string, service *RemoteService) {
	rsrvs, ok := self.methodRemoteServices[method]
	if !ok {
		rsrvs = make([]*RemoteService, 0)
	}
	self.methodRemoteServices[method] = append(rsrvs, service)
}

func (self *Router) RemoveRemote(method string, service *RemoteService) (changed bool) {
	if srvs, ok := self.methodRemoteServices[method]; ok {
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
					log.Panicf("remote services list has only 1 elements but found %d is not 0, method=%s", found, method)
				}
				delete(self.methodRemoteServices, method)
			} else {
				// removed srvs[found]
				self.methodRemoteServices[method] = append(srvs[:found], srvs[found+1:]...)
			}
			return true
		}

	}
	return false
}

func (self *Router) UpdateRemoteService(service *RemoteService, removed []string, added []string) {
	for _, mname := range removed {
		self.RemoveRemote(mname, service)
	}
	for _, mname := range added {
		self.AddRemote(mname, service)
	}
}

func (self *Router) SelectRemoteService(method string) (*RemoteService, bool) {
	if rsrvs, ok := self.methodRemoteServices[method]; ok && len(rsrvs) > 0 {
		idx := rand.Intn(len(rsrvs))
		return rsrvs[idx], true
	}
	return nil, false
}

func (self *Router) RemoteMethods() []string {
	methods := []string{}
	for mname, _ := range self.methodRemoteServices {
		methods = append(methods, mname)
	}
	return methods
}
