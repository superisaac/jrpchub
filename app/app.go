package rpcmapapp

import (
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	app     *App
	appOnce sync.Once
)

func GetApp() *App {
	appOnce.Do(func() {
		app = &App{
			Config: &AppConfig{},
		}
	})
	return app
}

func (self *App) GetRouter(ns string) *Router {
	if v, ok := self.routers.Load(ns); ok {
		router, _ := v.(*Router)
		return router
	} else {
		v, loaded := self.routers.LoadOrStore(ns, NewRouter(ns))
		if loaded {
			log.Warnf("routers concurrent load")
		}
		router, _ := v.(*Router)
		router.Start()
		return router
	}
}
