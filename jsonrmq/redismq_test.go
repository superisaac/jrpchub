package jsonrmq

import (
	"context"
	"fmt"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superisaac/jsonz"
	"io/ioutil"
	"os"
	"testing"
	"reflect"
)

func redisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	fmt.Printf("got redis addr %s \n", addr)
	opts := &redis.Options{
		Addr: addr,
		DB:   0,
	}
	return redis.NewClient(opts)
}

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestRedisMQ(t *testing.T) {
	assert := assert.New(t)

	c := redisClient()
	ctx := context.Background()
	ntf0 := jsonz.NewNotifyMessage("pos.change", []interface{}{100, 200})
	id0, err := Append(ctx, c, "testing", ntf0)
	if err != nil {
		fmt.Printf("append error %s %s\n", reflect.TypeOf(err), err)
	}
	assert.Nil(err)

	rng, err := GetTailRange(ctx, c, "testing", 1)
	assert.Nil(err)
	assert.Equal(1, len(rng.Items))
	assert.Equal(id0, rng.NextID)
	assert.Equal("Notify", rng.Items[0].Kind)
	assert.Equal("pos.change", rng.Items[0].Brief)

	ntf10 := rng.Items[0].Notify()
	assert.True(ntf10.IsNotify())
	assert.Equal("pos.change", ntf10.MustMethod())
	assert.Equal(json.Number("100"), ntf10.MustParams()[0])

}
