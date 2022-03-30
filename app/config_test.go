package app

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	cfgdata := `
---
mq:
  url: redis://127.0.0.1:6379/2
`

	appcfg := &AppConfig{}
	appcfg.LoadYamldata([]byte(cfgdata))
	u := appcfg.MQ.URL()
	assert.Equal("redis://127.0.0.1:6379/2", u.String())
	assert.Equal("redis", u.Scheme)
	assert.Equal("/2", u.Path)
	assert.NotNil(appcfg.MQ.url)
}
