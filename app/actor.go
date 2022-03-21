package rpczapp

import (
	"context"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"net/http"
)

func extractNamespace(ctx context.Context) string {
	if v := ctx.Value("authInfo"); v != nil {
		authInfo, _ := v.(*jsonzhttp.AuthInfo)
		if authInfo.Settings != nil {
			if nv, ok := authInfo.Settings["namespace"]; ok {
				if ns, ok := nv.(string); ok {
					return ns
				}
			}

		}

	}
	return "default"
}

func NewActor() *jsonzhttp.Actor {
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

	actor.OnMissing(func(req *jsonzhttp.RPCRequest) (interface{}, error) {
		ns := extractNamespace(req.HttpRequest().Context())
		msg := req.Msg()
		router := GetRouter(ns)
		return router.Feed(msg)
	})

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		ns := extractNamespace(r.Context())
		router := GetRouter(ns)
		router.DismissService(session.SessionID())
	})
	return actor
}
