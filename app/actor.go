package rpczapp

import (
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/jsonz/schema"
	"net/http"
)

func NewActor() *jsonzhttp.Actor {
	actor := jsonzhttp.NewActor()
	actor.OnTyped("rpcz.register", func(req *jsonzhttp.RPCRequest, methods []string) (string, error) {
		session := req.Session()
		if session == nil {
			return "", jsonz.ErrMethodNotFound
		}
		router := GetRouter()
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
		msg := req.Msg()
		router := GetRouter()
		return router.Feed(msg)
	})

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		router := GetRouter()
		router.RemoveService(session.SessionID())
	})
	return actor
}
