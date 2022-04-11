package app

import (
	"context"
	"fmt"
	"github.com/superisaac/jlib"
	"github.com/superisaac/jlib/http"
	"github.com/superisaac/jlib/schema"
	"github.com/superisaac/jrpchub/mq"
	"net/http"
)

const (
	declareSchema = `
---
type: method
description: declare serve methods, only callable via stream requests
params:
  - anyOf:
    - type: object
      properties: {}
    - type: "null"
`
	showSchemaSchema = `
---
type: method
description: show the schema of a method
params:
  - type: string
    name: method
`
	listMethodsSchema = `
---
type: method
description: list the names of provided methods and remote methods
params: []
returns:
  type: object
  properties:
    methods:
      type: list
      description: method names
      items: string
    remotes:
      type: list
      description: remote method names
      items: string
`
)

func extractNamespace(ctx context.Context) string {
	if authinfo, ok := jlibhttp.AuthInfoFromContext(ctx); ok && authinfo != nil {
		if nv, ok := authinfo.Settings["namespace"]; ok {
			if ns, ok := nv.(string); ok {
				return ns
			}
		}
	}
	return "default"
}

func NewActor(apps ...*App) *jlibhttp.Actor {
	var app *App
	for _, a := range apps {
		app = a
	}
	if app == nil {
		app = Application()
	}

	actor := jlibhttp.NewActor()

	if !app.Config.MQ.Empty() {
		mqactor := jrpchubmq.NewActor(app.Config.MQ.URL())
		actor.AddChild(mqactor)
	}

	// declare methods
	actor.OnTyped("rpc.declare", func(req *jlibhttp.RPCRequest, methods map[string]interface{}) (string, error) {
		session := req.Session()
		if session == nil {
			return "", jlib.ErrMethodNotFound
		}
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		service := router.GetService(session)

		methodSchemas := map[string]jlibschema.Schema{}
		for mname, smap := range methods {
			if !jlib.IsPublicMethod(mname) {
				continue
			}
			if smap == nil {
				methodSchemas[mname] = nil
			} else {
				builder := jlibschema.NewSchemaBuilder()
				s, err := builder.Build(smap)
				if err != nil {
					return "", jlib.ParamsError(fmt.Sprintf("schema of %s build failed", mname))
				}
				methodSchemas[mname] = s
			}
		}
		err := service.UpdateMethods(methodSchemas)
		if err != nil {
			return "", err
		}
		return "ok", nil
	}, jlibhttp.WithSchemaYaml(declareSchema))

	// list the methods the current node can provide, the remote methods are also listed
	actor.On("rpc.methods", func(req *jlibhttp.RPCRequest, params []interface{}) (interface{}, error) {
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		methods := []string{}
		methods = append(methods, actor.MethodList()...)
		methods = append(methods, router.ServingMethods()...)
		remote_methods := router.RemoteMethods()

		r := map[string]interface{}{
			"methods": methods,
			"remotes": remote_methods,
		}
		return r, nil
	}, jlibhttp.WithSchemaYaml(listMethodsSchema))

	actor.OnTyped("rpc.schema", func(req *jlibhttp.RPCRequest, method string) (map[string]interface{}, error) {
		// from actor
		if actor.Has(method) {
			if schema, ok := actor.GetSchema(method); ok {
				return schema.Map(), nil
			} else {
				return nil, jlib.ParamsError("no schema")
			}
		}

		// get schema from router
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		if srv, ok := router.SelectService(method); ok {
			if schema, ok := srv.GetSchema(method); ok {
				return schema.Map(), nil
			} else {
				return nil, jlib.ParamsError("no schema")
			}
		}

		return nil, jlib.ParamsError("no schema")
	}, jlibhttp.WithSchemaYaml(showSchemaSchema))

	actor.OnMissing(func(req *jlibhttp.RPCRequest) (interface{}, error) {
		msg := req.Msg()
		ns := extractNamespace(req.Context())

		router := app.GetRouter(ns)
		return router.Feed(msg)
	})

	actor.OnClose(func(r *http.Request, session jlibhttp.RPCSession) {
		ns := extractNamespace(r.Context())
		router := app.GetRouter(ns)
		router.DismissService(session.SessionID())
	})
	return actor
}
