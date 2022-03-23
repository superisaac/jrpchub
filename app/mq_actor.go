package rpczapp

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcz/jsonrmq"
	"net/http"
	"sync"
)

type subscription struct {
	subID      string
	context    context.Context
	cancelFunc func()
}

func NewMQActor(mqurl string) *jsonzhttp.Actor {
	actor := jsonzhttp.NewActor()

	subscriptions := sync.Map{}

	// currently only support redis
	log.Infof("create mq actor, currently only redis mq is supported")
	rdb, err := NewRedisClient(mqurl)
	if err != nil {
		log.Panicf("create redis client error %s", err)
	}

	actor.OnTyped("rpczmq.get", func(req *jsonzhttp.RPCRequest, prevID string, count int) (map[string]interface{}, error) {
		ns := extractNamespace(req.HttpRequest().Context())
		rng, err := jsonrmq.GetRange(
			req.Context(),
			rdb, ns, prevID, int64(count))
		if err != nil {
			return nil, err
		}
		return rng.JsonResult(), err
	})

	actor.OnTyped("rpczmq.tail", func(req *jsonzhttp.RPCRequest, count int) (map[string]interface{}, error) {
		ns := extractNamespace(req.HttpRequest().Context())
		rng, err := jsonrmq.GetTailRange(
			req.Context(),
			rdb, ns, int64(count))
		if err != nil {
			return nil, err
		}
		return rng.JsonResult(), err
	})

	actor.On("rpczmq.add", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
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

	actor.On("rpczmq.subscribe", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
		session := req.Session()
		if session == nil {
			return nil, jsonz.ErrMethodNotFound
		}
		ns := extractNamespace(req.HttpRequest().Context())
		var mths []string
		err := jsonz.DecodeInterface(params, &mths)
		if err != nil {
			log.Warnf("decode methods %s", err)
			return nil, err
		}
		methods := map[string]bool{}

		for _, method := range mths {
			methods[method] = true
		}

		ctx, cancel := context.WithCancel(req.HttpRequest().Context())
		sub := &subscription{
			subID:      jsonz.NewUuid(),
			context:    ctx,
			cancelFunc: cancel,
		}
		subscriptions.Store(session.SessionID(), sub)
		log.Infof("subscription %s created", sub.subID)

		go func() {
			err := jsonrmq.Subscribe(ctx, rdb, ns, func(item jsonrmq.MQItem) {
				if _, ok := methods[item.Brief]; !ok {
					return
				}
				ntf := item.Notify()
				ntfmap, err := jsonz.MessageMap(ntf)
				if err != nil {
					panic(err)
				}
				params := map[string]interface{}{
					"subscription": sub.subID,
					"mqID":         item.ID,
					"msg":          ntfmap,
				}
				itemntf := jsonz.NewNotifyMessage("rpcz.item", params)
				session.Send(itemntf)
			})
			if err != nil {
				log.Errorf("subscribe error %s", err)
			}
		}()
		return sub.subID, nil
	}) // end of on rpczmq.subscribe

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		if v, ok := subscriptions.LoadAndDelete(session.SessionID()); ok {
			sub, _ := v.(*subscription)
			log.Infof("cancel subscription %s", sub.subID)
			sub.cancelFunc()
		}
	})
	return actor
}
