package playbook

import (
	log "github.com/sirupsen/logrus"
	"github.com/superisaac/jlib/schema"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

func (self *PlaybookConfig) Load(filePath string) error {
	log.Infof("read playbook from %s", filePath)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	return self.LoadBytes(data)
}

func (self *PlaybookConfig) LoadBytes(data []byte) error {
	err := yaml.Unmarshal(data, self)
	if err != nil {
		return err
	}
	err = self.validateValues()
	return err
}

func (self *PlaybookConfig) validateValues() error {
	if self.Version == "" {
		self.Version = "1.0"
	}

	for _, method := range self.Methods {
		if method.SchemaInterface != nil {
			builder := jlibschema.NewSchemaBuilder()
			s, err := builder.BuildYamlInterface(method.SchemaInterface)
			if err != nil {
				return err
			}
			method.innerSchema = s
			method.SchemaInterface = s.RebuildType()
		}
	}
	return nil
}
