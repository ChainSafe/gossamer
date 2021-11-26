// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func Test_marshalYAML(t *testing.T) {
	var mustMarshal = func(c conf) (yml []byte) {
		yml, err := yaml.Marshal(c)
		if err != nil {
			panic(err)
		}
		return
	}
	type args struct {
		opts options
	}
	tests := []struct {
		name    string
		args    args
		wantYml []byte
		wantErr bool
	}{
		{
			name: "zero case options",
			args: args{opts: options{}},
			wantYml: mustMarshal(
				conf{
					Instances: []instance{
						{
							PrometheusURL: "http://127.0.0.1:9876/metrics",
							Metrics: []string{
								"gossamer_*",
								"network_*",
								"service_*",
								"system_*",
							},
							HealthServiceCheck: true,
						},
					},
				},
			),
		},
		{
			name: "options with ns and tags",
			args: args{opts: options{
				Namespace: "SomeNamespace",
				Tags:      []string{"some", "tags"},
			}},
			wantYml: mustMarshal(
				conf{
					Instances: []instance{
						{
							PrometheusURL: "http://127.0.0.1:9876/metrics",
							Metrics: []string{
								"gossamer_*",
								"network_*",
								"service_*",
								"system_*",
							},
							HealthServiceCheck: true,
							Namespace:          "SomeNamespace",
							Tags:               []string{"some", "tags"},
						},
					},
				},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYml, err := marshalYAML(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotYml, tt.wantYml) {
				t.Errorf("marshalYAML() = %s, want %s", gotYml, tt.wantYml)
			}
		})
	}
}
