// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/internal/httpserver"
	"github.com/stretchr/testify/assert"
)

func TestServer_Start(t *testing.T) {
	type fields struct {
		server *httpserver.Server
	}
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				server: httpserver.New("metrics", ":0", http.NewServeMux(), logger),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				server: tt.fields.server,
			}
			if err := s.Start(tt.args.address); (err != nil) != tt.wantErr {
				t.Errorf("Server.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
			err := s.Stop()
			if err != nil {
				t.Errorf("unexpected error after stopping: %v", err)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name  string
		args  args
		wantS *Server
	}{
		{
			name: "happy path",
			args: args{
				address: "someAddress",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.args.address)
			assert.NotNil(t, s)
		})
	}
}

func TestNewIntervalConfig(t *testing.T) {
	type args struct {
		publish bool
	}
	tests := []struct {
		name string
		args args
		want IntervalConfig
	}{
		{
			name: "happy path",
			args: args{
				publish: true,
			},
			want: IntervalConfig{
				Publish:  true,
				Interval: defaultInterval,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewIntervalConfig(tt.args.publish); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIntervalConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
