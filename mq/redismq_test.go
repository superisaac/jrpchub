package mq

import (
	"context"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superisaac/jsoff"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestRedisMQ(t *testing.T) {
	assert := assert.New(t)

	mqurl, err := url.Parse("redis://localhost:6379/7")
	assert.Nil(err)

	mc := NewRedisMQClient(mqurl)
	ctx := context.Background()
	ntf0 := jsoff.NewNotifyMessage("pos.change", []interface{}{100, 200})
	id0, err := mc.Add(ctx, "testing", ntf0)
	assert.Nil(err)

	chunk, err := mc.Tail(ctx, "testing", 1)
	assert.Nil(err)
	assert.Equal(1, len(chunk.Items))
	assert.Equal(id0, chunk.LastOffset)
	assert.Equal("Notify", chunk.Items[0].Kind)
	assert.Equal("pos.change", chunk.Items[0].Brief)

	ntf10 := chunk.Items[0].Notify()
	assert.True(ntf10.IsNotify())
	assert.Equal("pos.change", ntf10.MustMethod())
	assert.Equal(json.Number("100"), ntf10.MustParams()[0])

}
