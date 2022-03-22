package jsonrmq

import (
	"context"
	"encoding/json"
        log "github.com/sirupsen/logrus"
	"github.com/go-redis/redis/v8"	
        "github.com/stretchr/testify/assert"
	"github.com/superisaac/jsonz"
        "io/ioutil"
        "os"
        "testing"
)

func redisClient() *redis.Client {
	addr :=  os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	opts := &redis.Options{
                Addr:    addr,
                DB:       6,
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
	msg0 := jsonz.NewNotifyMessage("pos.change", []interface{}{100, 200})
	id0, err := Append(ctx, c, "testing", msg0)
	assert.Nil(err)
	
	rng, err := GetTailRange(ctx, c, "testing", 1)
	assert.Nil(err)
	assert.Equal(1, len(rng.Items))
	assert.Equal(id0, rng.NextID)
	assert.Equal("Notify", rng.Items[0].Kind)
	assert.Equal("pos.change", rng.Items[0].Brief)
	
	msg10, err := jsonz.ParseBytes(rng.Items[0].Body)
	assert.Nil(err)
	assert.True(msg10.IsNotify())
	assert.Equal("pos.change", msg10.MustMethod())
	assert.Equal(json.Number("100"), msg10.MustParams()[0])
	
}


