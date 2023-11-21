package app

import (
	"context"
	"fmt"
	"github.com/superisaac/jsoff"
	"github.com/superisaac/jsoff/net"
	"github.com/superisaac/jsoff/schema"
	"github.com/superisaac/rpcmux/mq"
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
	if authinfo, ok := jsoffnet.AuthInfoFromContext(ctx); ok && authinfo != nil {
		if nv, ok := authinfo.Settings["namespace"]; ok {
			if ns, ok := nv.(string); ok {
				return ns
			}
		}
	}
	return "default"
}

func NewActor(apps ...*App) *jsoffnet.Actor {
	var app *App
	for _, a := range apps {
		app = a
	}
	if app == nil {
		app = Application()
	}

	actor := jsoffnet.NewActor()

	if !app.Config.MQ.Empty() {
		mqactor := mq.NewActor(app.Config.MQ.URL())
		actor.AddChild(mqactor)
	}

	// declare methods
	actor.OnTypedRequest("rpcmux.declare", func(req *jsoffnet.RPCRequest, methods map[string]interface{}) (string, error) {
		session := req.Session()
		if session == nil {
			return "", jsoff.ErrMethodNotFound
		}
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		service, _ := router.GetService(session)

		methodSchemas := map[string]jsoffschema.Schema{}
		for mname, smap := range methods {
			if !jsoff.IsPublicMethod(mname) {
				continue
			}
			if smap == nil {
				methodSchemas[mname] = nil
			} else {
				builder := jsoffschema.NewSchemaBuilder()
				s, err := builder.Build(smap)
				if err != nil {
					return "", jsoff.ParamsError(fmt.Sprintf("schema of %s build failed", mname))
				}
				methodSchemas[mname] = s
			}
		}
		err := service.UpdateMethods(methodSchemas)
		if err != nil {
			return "", err
		}
		return "ok", nil
	}, jsoffnet.WithSchemaYaml(declareSchema))

	// list the methods the current node can provide, the remote methods are also listed
	actor.OnRequest("rpcmux.methods", func(req *jsoffnet.RPCRequest, params []interface{}) (interface{}, error) {
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
	}, jsoffnet.WithSchemaYaml(listMethodsSchema))

	actor.OnTypedRequest("rpcmux.schema", func(req *jsoffnet.RPCRequest, method string) (map[string]interface{}, error) {
		// from actor
		if actor.Has(method) {
			if schema, ok := actor.GetSchema(method); ok {
				return schema.Map(), nil
			} else {
				return nil, jsoff.ParamsError("no schema")
			}
		}

		// get schema from router
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		if srv, ok := router.SelectService(method); ok {
			if schema, ok := srv.GetSchema(method); ok {
				return schema.Map(), nil
			} else {
				return nil, jsoff.ParamsError("no schema")
			}
		}

		return nil, jsoff.ParamsError("no schema")
	}, jsoffnet.WithSchemaYaml(showSchemaSchema))

	actor.OnMissing(func(req *jsoffnet.RPCRequest) (interface{}, error) {
		msg := req.Msg()
		ns := extractNamespace(req.Context())

		router := app.GetRouter(ns)
		return router.Feed(msg)
	})

	actor.OnClose(func(session jsoffnet.RPCSession) {
		ns := extractNamespace(session.Context())
		router := app.GetRouter(ns)
		router.DismissService(session.SessionID())
	})
	return actor
}
