package rpczapp

import (
	"context"
	//log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"net/http"
)

func extractNamespace(ctx context.Context) string {
	if v := ctx.Value("authInfo"); v != nil {
		authInfo, _ := v.(*jsonzhttp.AuthInfo)
		if authInfo != nil && authInfo.Settings != nil {
			if nv, ok := authInfo.Settings["namespace"]; ok {
				if ns, ok := nv.(string); ok {
					return ns
				}
			}

		}

	}
	return "default"
}

func NewActor(cfg *RPCZConfig) *jsonzhttp.Actor {
	if cfg == nil {
		cfg = &RPCZConfig{}
	}

	actor := jsonzhttp.NewActor()
	actor.OnTyped("rpcz.declare", func(req *jsonzhttp.RPCRequest, methods []string) (string, error) {
		session := req.Session()
		if session == nil {
			return "", jsonz.ErrMethodNotFound
		}
		ns := extractNamespace(req.HttpRequest().Context())
		router := GetRouter(ns)
		service := router.GetService(session)

		// TODO: build schema
		m := map[string]jsonzschema.Schema{}
		for _, mname := range methods {
			m[mname] = nil
		}
		service.UpdateMethods(m)
		return "ok", nil
	})

	var mqactor *jsonzhttp.Actor
	if cfg.MQUrl != "" {
		mqactor = NewMQActor(cfg.MQUrl)
	}

	actor.OnMissing(func(req *jsonzhttp.RPCRequest) (interface{}, error) {
		msg := req.Msg()
		if mqactor != nil && msg.IsRequestOrNotify() && mqactor.HasHandler(msg.MustMethod()) {
			return mqactor.Feed(req)
		}

		ns := extractNamespace(req.HttpRequest().Context())

		router := GetRouter(ns)
		return router.Feed(msg)
	})

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		ns := extractNamespace(r.Context())
		router := GetRouter(ns)
		if dismissed := router.DismissService(session.SessionID()); !dismissed {
			if mqactor != nil {
				mqactor.HandleClose(r, session)
			}
		}
	})
	return actor
}
