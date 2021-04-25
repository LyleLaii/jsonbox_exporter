package config

import (
	"io/ioutil"

	pconfig "github.com/prometheus/common/config"
	"gopkg.in/yaml.v2"
)

type ModuleConfig struct {
	RequestConfig Request  `yaml:"request"`
	Metrics       []Metric `yaml:metrics`
}

type Request struct {
	Method string `yaml:"method,omitempty"` // Not used now, save for the future
	Headers map[string]string `yaml:"headers,omitempty"'`
	ClientConfig pconfig.HTTPClientConfig `yaml:"client_config,omitempty"`
	Params map[string]string `yaml:"params,omitempty"`
	Body struct {
		Content    string `yaml:"content"`
		Templatize bool   `yaml:"templatize,omitempty"`
	} `yaml:"body,omitempty"`
}

// Metric contains values that define a metric
type Metric struct {
	Name   string
	Path   string
	Labels map[string]string
	Type   MetricType
	Help   string
	Values map[string]string
}

type MetricType string

const (
	ValueScrape  MetricType = "value" // default
	ObjectScrape MetricType = "object"
)

// Config contains metrics and headers defining a configuration
type Config struct {
	Modules          map[string]*ModuleConfig        `yaml:"modules,omitempty"`
}

func LoadConfig(configPath string) (Config, error) {
	var config Config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	// Complete Defaults
	for _, v := range config.Modules {
		for i := 0; i < len(v.Metrics); i++ {
			if v.Metrics[i].Type == "" {
				v.Metrics[i].Type = ValueScrape
			}
			if v.Metrics[i].Help == "" {
				v.Metrics[i].Help = v.Metrics[i].Name
			}
		}
	}

	return config, nil
}
