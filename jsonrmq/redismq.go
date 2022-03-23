package jsonrmq

// currently we use redis
import (
	//"fmt"
	"context"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jsonz"
	"time"
)

func streamsKey(section string) string {
	return "jsonrmq:" + section
}

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

func Add(ctx context.Context, rdb *redis.Client, section string, ntf *jsonz.NotifyMessage) (string, error) {
	kind := "Notify"
	brief := ntf.MustMethod()
	values := map[string]interface{}{
		"kind":    kind,
		"brief":   brief,
		"msgdata": jsonz.MessageString(ntf),
	}
	addedID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamsKey(section),
		Values: values,
		MaxLen: 10000,
	}).Result()
	return addedID, err
}

func GetRange(ctx context.Context, rdb *redis.Client, section string, prevID string, count int64) (MQRange, error) {
	if count <= 0 {
		log.Panicf("count %d <= 0", count)
	}
	skey := streamsKey(section)
	if prevID == "" {
		// get the last item
		xmsgs, err := rdb.XRevRangeN(ctx, skey, "+", "-", 1).Result()
		if err != nil {
			return MQRange{}, err
		}
		// assert len(msgs) <= 1
		if len(xmsgs) > 1 {
			log.Panicf("xrevrange(%s, +, -, 1) got more than 1 items", skey)
		}
		return ConvertXMsgs(xmsgs, prevID, true), nil
	} else {
		xmsgs, err := rdb.XRangeN(ctx, skey, "("+prevID, "+", count).Result()
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

	revmsgs, err := rdb.XRevRangeN(ctx, streamsKey(section), "+", "-", count).Result()
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

func Subscribe(rootctx context.Context, rdb *redis.Client, section string, callback func(item MQItem)) error {
	ctx, cancel := context.WithCancel(rootctx)

	defer func() {
		log.Info("subscribe stop")
		cancel()
	}()

	prevID := ""
	for {
		rng, err := GetRange(rootctx, rdb, section, prevID, 10)
		if err != nil {
			return err
		}
		prevID = rng.NextID
		if len(rng.Items) > 0 {
			log.Infof("got range of %d items, nextID=%s", len(rng.Items), rng.NextID)
			for _, item := range rng.Items {
				callback(item)
			}
		} else {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Millisecond):
				continue
			}
		}

	}
}
