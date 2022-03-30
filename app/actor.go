package rpcmapapp

import (
	"context"
	"fmt"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"github.com/superisaac/rpcmap/mq"
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
    remote:
      type: list
      description: remote method names
      items: string
`
)

func extractNamespace(ctx context.Context) string {
	if authinfo, ok := jsonzhttp.AuthInfoFromContext(ctx); ok {
		if nv, ok := authinfo.Settings["namespace"]; ok {
			if ns, ok := nv.(string); ok {
				return ns
			}
		}
	}
	return "default"
}

func NewActor() *jsonzhttp.Actor {
	app := GetApp()

	actor := jsonzhttp.NewActor()
	children := []*jsonzhttp.Actor{}

	if !app.Config.MQ.Empty() {
		mqactor := rpcmapmq.NewActor(app.Config.MQ.URL())
		children = append(children, mqactor)
	}

	// declare methods
	actor.OnTyped("rpc.declare", func(req *jsonzhttp.RPCRequest, methods map[string]interface{}) (string, error) {
		session := req.Session()
		if session == nil {
			return "", jsonz.ErrMethodNotFound
		}
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		service := router.GetService(session)

		methodSchemas := map[string]jsonzschema.Schema{}
		for mname, smap := range methods {
			if smap == nil {
				methodSchemas[mname] = nil
			} else {
				builder := jsonzschema.NewSchemaBuilder()
				s, err := builder.Build(smap)
				if err != nil {
					return "", jsonz.ParamsError(fmt.Sprintf("schema of %s build failed", mname))
				}
				methodSchemas[mname] = s
			}
		}
		err := service.UpdateMethods(methodSchemas)
		if err != nil {
			return "", err
		}
		return "ok", nil
	}, jsonzhttp.WithSchemaYaml(declareSchema))

	// list the methods the current node can provide, the remote methods are also listed
	actor.On("rpc.methods", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		methods := []string{}
		methods = append(methods, actor.MethodList()...)
		for _, child := range children {
			methods = append(methods, child.MethodList()...)
		}
		methods = append(methods, router.ServingMethods()...)

		remote_methods := router.RemoteMethods()

		r := map[string]interface{}{
			"methods": methods,
			"remote":  remote_methods,
		}
		return r, nil
	}, jsonzhttp.WithSchemaYaml(listMethodsSchema))

	actor.OnTyped("rpc.schema", func(req *jsonzhttp.RPCRequest, method string) (map[string]interface{}, error) {
		// from actor
		if actor.Has(method) {
			if schema, ok := actor.GetSchema(method); ok {
				return schema.RebuildType(), nil
			} else {
				return nil, jsonz.ParamsError("no schema")
			}
		}

		// from children
		for _, c := range children {
			if !c.Has(method) {
				continue
			}
			if schema, ok := c.GetSchema(method); ok {
				return schema.RebuildType(), nil
			} else {
				return nil, jsonz.ParamsError("no schema")
			}
		}

		// get schema from router
		ns := extractNamespace(req.Context())
		router := app.GetRouter(ns)
		if srv, ok := router.SelectService(method); ok {
			if schema, ok := srv.GetSchema(method); ok {
				return schema.RebuildType(), nil
			} else {
				return nil, jsonz.ParamsError("no schema")
			}
		}

		return nil, jsonz.ParamsError("no schema")
	}, jsonzhttp.WithSchemaYaml(showSchemaSchema))

	actor.OnMissing(func(req *jsonzhttp.RPCRequest) (interface{}, error) {
		msg := req.Msg()
		if msg.IsRequestOrNotify() {
			for _, child := range children {
				if child.Has(msg.MustMethod()) {
					return child.Feed(req)
				}
			}
		}

		ns := extractNamespace(req.Context())

		router := app.GetRouter(ns)
		return router.Feed(msg)
	})

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		ns := extractNamespace(r.Context())
		router := app.GetRouter(ns)
		if dismissed := router.DismissService(session.SessionID()); !dismissed {
			for _, child := range children {
				child.HandleClose(r, session)
			}
		}
	})
	return actor
}
