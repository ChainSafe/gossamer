package main

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func Test_marshalYAML(t *testing.T) {
	c := conf{
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
	}
	expected, err := yaml.Marshal(c)
	if err != nil {
		t.Errorf("%v", err)
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
			args:    args{opts: options{}},
			wantYml: expected,
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
