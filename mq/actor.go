package rpcmapmq

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"net/http"
	"net/url"
	"sync"
)

const (
	tailSchema = `
---
type: method
description: get the tail elements
params:
  - type: number
    name: count
    description: item count
`

	getSchema = `
---
type: method
description: get a range of mq 
params:
  - name: prevID
    type: string
    description: previous id, empty prevID means take the last item
  - name: count
    type: integer
    description: get count
`

	addSchema = `
---
type: method
description: mq.add add a notify methods
params:
  - name: notifymethod
    type: string
additionalParams:
  type: any
`
	subscribeSchema = `
---
type: method
description: mq.subscribe subscribe a stream of notify message
params: []
additionalParams:
  type: string
  name: followedMethod
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

type subscription struct {
	subID      string
	context    context.Context
	cancelFunc func()
}

func NewActor(mqurl *url.URL) *jsonzhttp.Actor {
	actor := jsonzhttp.NewActor()

	subscriptions := sync.Map{}

	// currently only support redis
	log.Infof("create mq actor, currently only redis mq is supported")

	mqclient := NewMQClient(mqurl)

	actor.OnTyped("mq.get", func(req *jsonzhttp.RPCRequest, prevID string, count int) (map[string]interface{}, error) {
		ns := extractNamespace(req.Context())
		chunk, err := mqclient.Chunk(
			req.Context(),
			ns, prevID, int64(count))
		if err != nil {
			return nil, err
		}
		return chunk.JsonResult(), err
	}, jsonzhttp.WithSchemaYaml(getSchema))

	actor.OnTyped("mq.tail", func(req *jsonzhttp.RPCRequest, count int) (map[string]interface{}, error) {
		ns := extractNamespace(req.Context())
		chunk, err := mqclient.Tail(
			req.Context(),
			ns, int64(count))
		if err != nil {
			return nil, err
		}
		return chunk.JsonResult(), err
	}, jsonzhttp.WithSchemaYaml(tailSchema))

	actor.On("mq.add", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
		if len(params) == 0 {
			return nil, jsonz.ParamsError("notify method not provided")
		}
		ns := extractNamespace(req.Context())

		method, ok := params[0].(string)
		if !ok {
			return nil, jsonz.ParamsError("method is not string")
		}

		ntf := jsonz.NewNotifyMessage(method, params[1:])
		id, err := mqclient.Add(req.Context(), ns, ntf)
		return id, err
	}, jsonzhttp.WithSchemaYaml(addSchema))

	actor.On("mq.subscribe", func(req *jsonzhttp.RPCRequest, params []interface{}) (interface{}, error) {
		session := req.Session()
		if session == nil {
			return nil, jsonz.ErrMethodNotFound
		}
		if _, ok := subscriptions.Load(session.SessionID()); ok {
			// this session already subscribed
			log.Warnf("mq.subscribe already called on session %s", session.SessionID())
			return nil, jsonz.ErrMethodNotFound
		}
		ns := extractNamespace(req.Context())
		var mths []string
		err := jsonz.DecodeInterface(params, &mths)
		if err != nil {
			log.Warnf("decode methods %s", err)
			return nil, err
		}

		followedMethods := map[string]bool{}
		for _, method := range mths {
			followedMethods[method] = true
		}

		ctx, cancel := context.WithCancel(req.Context())
		sub := &subscription{
			subID:      jsonz.NewUuid(),
			context:    ctx,
			cancelFunc: cancel,
		}
		subscriptions.Store(session.SessionID(), sub)
		log.Infof("subscription %s created", sub.subID)

		itemSub := make(chan MQItem, 100)

		go receiveItems(ctx, itemSub, session, sub, followedMethods)
		go func() {
			err := mqclient.Subscribe(ctx, ns, itemSub)
			if err != nil {
				log.Errorf("subscribe error %s", err)
			}
		}()
		return sub.subID, nil
	}, jsonzhttp.WithSchemaYaml(subscribeSchema)) // end of on mq.subscribe

	actor.OnClose(func(r *http.Request, session jsonzhttp.RPCSession) {
		if v, ok := subscriptions.LoadAndDelete(session.SessionID()); ok {
			sub, _ := v.(*subscription)
			log.Infof("cancel subscription %s", sub.subID)
			sub.cancelFunc()
		}
	})
	return actor
}

// receive items from channel and send them back to session
func receiveItems(
	rootCtx context.Context,
	itemSub chan MQItem,
	session jsonzhttp.RPCSession,
	sub *subscription,
	followedMethods map[string]bool) {

	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-itemSub:
			if !ok {
				log.Infof("item sub ended, just return")
				return
			}
			if len(followedMethods) > 0 {
				if _, ok := followedMethods[item.Brief]; !ok {
					continue
				}
			}
			ntf := item.Notify()
			ntfmap, err := jsonz.MessageMap(ntf)
			if err != nil {
				panic(err)
			}
			params := map[string]interface{}{
				"subscription": sub.subID,
				"offset":       item.Offset,
				"msg":          ntfmap,
			}
			itemntf := jsonz.NewNotifyMessage("rpcmap.item", params)
			session.Send(itemntf)
		}
	}
}
