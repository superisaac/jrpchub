package app

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	app     *App
	appOnce sync.Once
)

func Application() *App {
	appOnce.Do(func() {
		app = NewApp()
	})
	return app
}

func NewApp(rootCtxs ...context.Context) *App {
	var rootCtx context.Context = context.Background()
	for _, rctx := range rootCtxs {
		rootCtx = rctx
	}
	ctx, cancel := context.WithCancel(rootCtx)
	return &App{
		Config:     &AppConfig{},
		ctx:        ctx,
		cancelFunc: cancel,
	}
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
		router.app = self
		router.Start()
		return router
	}
}

func (self *App) Context() context.Context {
	return self.ctx
}

func (self *App) Stop() {
	self.cancelFunc()
}
