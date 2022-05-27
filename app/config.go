package app

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/url"
	"os"
)

// MQConfig
func (self MQConfig) Empty() bool {
	return self.Urlstr == ""
}

func (self *MQConfig) URL() *url.URL {
	if self.url == nil {
		u, err := url.Parse(self.Urlstr)
		if err != nil {
			panic(err)
		}
		self.url = u
	}
	return self.url
}

func (self *MQConfig) validateValues() error {
	u, err := url.Parse(self.Urlstr)
	if err != nil {
		return errors.Wrap(err, "url.Parse")
	}
	if u.Scheme != "redis" {
		return errors.New("url scheme is not redis")
	}
	self.url = u
	return nil
}

func (self *AppConfig) Load(yamlPath string) error {
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		if err != nil {
			return errors.Wrap(err, "os.Stat")
		}
		return nil
	}

	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return errors.Wrap(err, "ioutil.ReadFile")
	}
	return self.LoadYamldata(data)
}

func (self *AppConfig) LoadYamldata(yamlData []byte) error {
	err := yaml.Unmarshal(yamlData, self)
	if err != nil {
		return errors.Wrap(err, "yaml.Unmarshal")
	}
	return self.validateValues()
}

func (self *AppConfig) validateValues() error {
	if self.Server.TLS != nil {
		err := self.Server.TLS.ValidateValues()
		if err != nil {
			return err
		}
	}
	if self.Server.Auth != nil {
		err := self.Server.Auth.ValidateValues()
		if err != nil {
			return err
		}
	}

	if !self.MQ.Empty() {
		err := self.MQ.validateValues()
		if err != nil {
			return err
		}
	}

	return nil
}
