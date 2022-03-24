package rpczmq

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
)

// mq item
func (self MQItem) Notify() *jsonz.NotifyMessage {
	msg, err := jsonz.ParseBytes(self.MsgData)
	if err != nil {
		log.Panicf("parse item bytes %s", err)
	}
	return msg.(*jsonz.NotifyMessage)
}

// mq range
func (self MQChunk) JsonResult() map[string]interface{} {
	itemmaps := make([]map[string]interface{}, 0)
	for _, item := range self.Items {
		ntf := item.Notify()
		ntf.SetTraceId(item.ID)
		ntfmap, err := jsonz.MessageMap(ntf)
		if err != nil {
			panic(err)
		}

		itemmap := map[string]interface{}{
			"mqID": item.ID,
			"msg":  ntfmap,
		}
		itemmaps = append(itemmaps, itemmap)
	}
	return map[string]interface{}{
		"items":  itemmaps,
		"nextID": self.NextID,
	}
}

func NewMQClient(mqurl string) MQClient {
	// TODO: more MQ type
	return NewRedisMQClient(mqurl)
}
