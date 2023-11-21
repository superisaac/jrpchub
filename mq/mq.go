package mq

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsoff"
	"net/url"
)

// mq item
func (self MQItem) Notify() *jsoff.NotifyMessage {
	msg, err := jsoff.ParseBytes(self.MsgData)
	if err != nil {
		log.Panicf("parse item bytes %s", err)
	}
	return msg.(*jsoff.NotifyMessage)
}

// mq range
func (self MQChunk) JsonResult() map[string]interface{} {
	itemmaps := make([]map[string]interface{}, 0)
	for _, item := range self.Items {
		ntf := item.Notify()
		ntfmap, err := jsoff.MessageMap(ntf)
		if err != nil {
			panic(err)
		}

		itemmap := map[string]interface{}{
			"offset": item.Offset,
			"msg":    ntfmap,
		}
		itemmaps = append(itemmaps, itemmap)
	}
	return map[string]interface{}{
		"items":      itemmaps,
		"lastoffset": self.LastOffset,
	}
}

func NewMQClient(mqurl *url.URL) MQClient {
	// TODO: more MQ type
	return NewRedisMQClient(mqurl)
}
