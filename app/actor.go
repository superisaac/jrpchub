package rpczapp

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"github.com/superisaac/rpcz/jsonrmq"
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

	if cfg.RedisMQUrl != "" {
		// has redis url
		log.Infof("redis mq exists")
		rdb, err := NewRedisClient(cfg.RedisMQUrl)
		if err != nil {
			panic(err)
		}

		actor.OnTyped("redismq.get", func(req *jsonzhttp.RPCRequest, prevID string, count int) (map[string]interface{}, error) {
			ns := extractNamespace(req.HttpRequest().Context())
			rng, err := jsonrmq.GetRange(
				req.Context(),
				rdb, ns, prevID, int64(count))
			if err != nil {
				return nil, err
			}
			return rng.JsonResult(), err
		})

		actor.OnTyped("redismq.tail", func(req *jsonzhttp.RPCRequest, count int) (map[string]interface{}, error) {
			ns := extractNamespace(req.HttpRequest().Context())
			rng, err := jsonrmq.GetTailRange(
				req.Context(),
				rdb, ns, int64(count))
			if err != nil {
				return nil, err
			}
			return rng.JsonResult(), err
		})

		actor.On("redismq.add", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
			if len(params) == 0 {
				return nil, jsonz.ParamsError("notify method not provided")
			}
			ns := extractNamespace(req.HttpRequest().Context())

			method, ok := params[0].(string)
			if !ok {
				return nil, jsonz.ParamsError("method is not string")
			}

			ntf := jsonz.NewNotifyMessage(method, params[1:])
			id, err := jsonrmq.Add(req.Context(), rdb, ns, ntf)
			return id, err
		})
	}

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
