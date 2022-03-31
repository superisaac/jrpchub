package playbook

import (
	"github.com/superisaac/jsonz/schema"
)

type ShellT struct {
	Cmd     string   `yaml:"command"`
	Env     []string `yaml:"env,omitempty"`
	Timeout *int     `yaml:"timeout,omitempty"`
}

type MethodT struct {
	Description     string             `yaml:"description,omitempty"`
	SchemaInterface interface{}        `yaml:"schema,omitempty"`
	Shell           *ShellT            `yaml:"shell,omitempty"`
	innerSchema     jsonzschema.Schema `yaml:"-"`
}

type PlaybookConfig struct {
	Version string                `yaml:"version,omitempty"`
	Methods map[string](*MethodT) `yaml:"methods,omitempty"`
}

type Playbook struct {
	Config PlaybookConfig
}
