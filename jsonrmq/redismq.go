package jsonrmq

// currently we use redis
import (
	//"fmt"
	"context"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
)

func (self MQItem) Notify() *jsonz.NotifyMessage {
	msg, err := jsonz.ParseBytes(self.MsgData)
	if err != nil {
		log.Panicf("parse item bytes %s", err)
	}
	return msg.(*jsonz.NotifyMessage)
}

func (self MQRange) JsonResult() map[string]interface{} {
	itemmaps := make([]map[string]interface{}, 0)
	for _, item := range self.Items {
		ntf := item.Notify()
		ntf.SetTraceId(item.ID)
		ntfmap, err := jsonz.MessageMap(ntf)
		if err != nil {
			panic(err)
		}

		// itemmap := map[string]interface{}{
		// 	"msg":  ntfmap,
		// 	"mqID": item.ID,
		// }
		ntfmap["mqID"] = item.ID
		itemmaps = append(itemmaps, ntfmap)
	}
	return map[string]interface{}{
		"items":  itemmaps,
		"nextID": self.NextID,
	}
}

func xmsgStr(xmsg *redis.XMessage, key string) string {
	if v, ok := xmsg.Values[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func ConvertXMsgs(xmsgs []redis.XMessage, defaultNextID string, onlyNextID bool) MQRange {
	items := []MQItem{}
	nextID := defaultNextID
	for _, xmsg := range xmsgs {
		nextID = xmsg.ID
		kind := xmsgStr(&xmsg, "kind")
		if kind == "" {
			continue
		}
		item := MQItem{
			ID:      xmsg.ID,
			Kind:    kind,
			Brief:   xmsgStr(&xmsg, "brief"),
			MsgData: []byte(xmsgStr(&xmsg, "msgdata")),
		}
		items = append(items, item)
	}

	if onlyNextID {
		items = []MQItem{}
	}
	return MQRange{
		Items:  items,
		NextID: nextID,
	}
}

func Add(ctx context.Context, rdb *redis.Client, section string, ntf jsonz.Message) (string, error) {
	streamsKey := "mq:" + section
	kind := "Notify"
	//var brief string
	brief := ntf.MustMethod()
	// if msg.IsRequest() {
	// 	kind = "Request"
	// 	brief = msg.MustMethod()
	// } else if msg.IsNotify() {
	// 	kind = "Notify"
	// 	brief = msg.MustMethod()
	// } else if msg.IsError() {
	// 	kind = "Error"
	// 	brief = fmt.Sprintf("%s", msg.MustId())
	// } else {
	// 	// msg.IsResult
	// 	kind = "Result"
	// 	brief = fmt.Sprintf("%s", msg.MustId())
	// }
	values := map[string]interface{}{
		"kind":    kind,
		"brief":   brief,
		"msgdata": jsonz.MessageString(ntf),
	}
	addedID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamsKey,
		Values: values,
		MaxLen: 10000,
	}).Result()
	return addedID, err
}

func GetRange(ctx context.Context, rdb *redis.Client, section string, prevID string, count int64) (MQRange, error) {
	if count <= 0 {
		log.Panicf("count %d <= 0", count)
	}
	streamsKey := "mq:" + section
	if prevID == "" {
		// get the last item
		xmsgs, err := rdb.XRevRangeN(ctx, streamsKey, "+", "-", 1).Result()
		if err != nil {
			return MQRange{}, err
		}
		// assert len(msgs) <= 1
		if len(xmsgs) > 1 {
			log.Panicf("xrevrange(%s, +, -, 1) got more than 1 items", streamsKey)
		}
		return ConvertXMsgs(xmsgs, prevID, true), nil
	} else {
		xmsgs, err := rdb.XRangeN(ctx, streamsKey, "("+prevID, "+", count).Result()
		if err != nil {
			return MQRange{}, err
		}
		return ConvertXMsgs(xmsgs, prevID, false), nil
	}
}

func GetTailRange(ctx context.Context, rdb *redis.Client, section string, count int64) (MQRange, error) {
	if count <= 0 {
		log.Panicf("count %d <= 0", count)
	}
	streamsKey := "mq:" + section

	revmsgs, err := rdb.XRevRangeN(ctx, streamsKey, "+", "-", count).Result()
	if err != nil {
		return MQRange{}, err
	}

	xmsgs := make([]redis.XMessage, len(revmsgs))
	// revert the list
	for i, xmsg := range revmsgs {
		xmsgs[len(revmsgs)-1-i] = xmsg
	}
	return ConvertXMsgs(xmsgs, "", false), nil
}
