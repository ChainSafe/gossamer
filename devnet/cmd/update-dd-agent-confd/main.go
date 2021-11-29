// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"log"

	_ "embed"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

type options struct {
	Namespace string   `short:"n" long:"namespace" description:"namespace that is prepended to all metrics" required:"true"` //nolint:lll
	Tags      []string `short:"t" long:"tags" description:"tags that are added to all metrics"`
}

func main() {
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panic(err)
	}
	yml, err := marshalYAML(opts)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("%s", yml)
}

//go:embed confd.yml
var confYAML string

func marshalYAML(opts options) (yml []byte, err error) {
	var c conf
	err = yaml.Unmarshal([]byte(confYAML), &c)
	if err != nil {
		return
	}

	c.Instances[0].Namespace = opts.Namespace
	c.Instances[0].Tags = opts.Tags

	return yaml.Marshal(c)
}

type instance struct {
	PrometheusURL      string   `yaml:"prometheus_url"`
	Namespace          string   `yaml:"namespace"`
	Metrics            []string `yaml:"metrics"`
	HealthServiceCheck bool     `yaml:"health_service_check"`
	Tags               []string `yaml:"tags,omitempty"`
}

type conf struct {
	InitConfig struct{}   `yaml:"init_config"`
	Instances  []instance `yaml:"instances"`
}
