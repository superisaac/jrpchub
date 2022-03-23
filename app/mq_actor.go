package rpczapp

import (
	//"context"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcz/jsonrmq"
)

func NewMQActor(mqurl string) *jsonzhttp.Actor {
	actor := jsonzhttp.NewActor()
	// currently only support redis
	log.Infof("create mq actor, currently only redis mq is supported")
	rdb, err := NewRedisClient(mqurl)
	if err != nil {
		log.Panicf("create redis client error %s", err)
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
	return actor
}
