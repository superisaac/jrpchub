package rpcmapworker

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superisaac/jsonz"
	"github.com/superisaac/jsonz/http"
	"github.com/superisaac/rpcmap/app"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestWorker(t *testing.T) {
	assert := assert.New(t)

	rootCtx := context.Background()

	// start rpcmap server
	actor := rpcmapapp.NewActor(nil)
	var handler http.Handler
	handler = jsonzhttp.NewGatewayHandler(rootCtx, actor, true)
	go jsonzhttp.ListenAndServe(rootCtx, "127.0.0.1:16001", handler)
	time.Sleep(100 * time.Millisecond)

	// prepare worker and connect to rpcmap server
	worker := NewServiceWorker([]string{"h2c://127.0.0.1:16001"})
	worker.OnTyped("echo", func(req *WorkerRequest, text string) (string, error) {
		return "echo: " + text, nil
	})
	go worker.ConnectWait(rootCtx)
	time.Sleep(100 * time.Millisecond)

	// create a request
	c, err := jsonzhttp.NewClient("http://127.0.0.1:16001")
	assert.Nil(err)

	reqmsg := jsonz.NewRequestMessage(1, "echo", []interface{}{"hi"})
	resmsg, err := c.Call(rootCtx, reqmsg)
	assert.Nil(err)
	assert.True(resmsg.IsResult())
	assert.Equal("echo: hi", resmsg.MustResult())
}
