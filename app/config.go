package rpcmapapp

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sync"
)

var (
	appcfg     *AppConfig
	appcfgOnce sync.Once
)

func GetAppConfig() *AppConfig {
	appcfgOnce.Do(func() {
		appcfg = &AppConfig{}
	})
	return appcfg
}

func (self *AppConfig) Load(yamlPath string) error {
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return nil
	}

	data, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return err
	}
	return self.LoadYamldata(data)
}

func (self *AppConfig) LoadYamldata(yamlData []byte) error {
	err := yaml.Unmarshal(yamlData, self)
	if err != nil {
		return err
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

	return nil
}

func (self AppConfig) MQAvailable() bool {
	return self.MQ.Url != ""
}
