package rpczmq

import (
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superisaac/jsonz"
	"io/ioutil"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestRedisMQ(t *testing.T) {
	assert := assert.New(t)

	mc := NewRedisMQClient("redis://localhost:6379/7")
	ctx := context.Background()
	ntf0 := jsonz.NewNotifyMessage("pos.change", []interface{}{100, 200})
	id0, err := mc.Add(ctx, "testing", ntf0)
	assert.Nil(err)

	rng, err := mc.Tail(ctx, "testing", 1)
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
