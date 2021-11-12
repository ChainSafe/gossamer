// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "dev default",
			want: &Config{
				Global: GlobalConfig{
					ID: "dev",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DevConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestGssmrConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "gossamer default",
			want: &Config{
				Global: GlobalConfig{
					ID: "gssmr",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GssmrConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestKusamaConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "kusama default",
			want: &Config{
				Global: GlobalConfig{
					ID: "ksmcc3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KusamaConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestPolkadotConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "polkadot default",
			want: &Config{
				Global: GlobalConfig{
					ID: "polkadot",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PolkadotConfig()
			assert.Equal(t, tt.want.Global.ID, got.Global.ID)
		})
	}
}

func TestRPCConfig_isRPCEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
		Enabled          bool
		External         bool
		Unsafe           bool
		UnsafeExternal   bool
		Port             uint32
		Host             string
		Modules          []string
		WSPort           uint32
		WS               bool
		WSExternal       bool
		WSUnsafe         bool
		WSUnsafeExternal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			want: false,
		},
		{
			name:   "enabled true",
			fields: fields{Enabled: true},
			want:   true,
		},
		{
			name:   "external true",
			fields: fields{External: true},
			want:   true,
		},
		{
			name:   "unsafe true",
			fields: fields{Unsafe: true},
			want:   true,
		},
		{
			name:   "unsafe external true",
			fields: fields{UnsafeExternal: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCConfig{
				Enabled:          tt.fields.Enabled,
				External:         tt.fields.External,
				Unsafe:           tt.fields.Unsafe,
				UnsafeExternal:   tt.fields.UnsafeExternal,
				Port:             tt.fields.Port,
				Host:             tt.fields.Host,
				Modules:          tt.fields.Modules,
				WSPort:           tt.fields.WSPort,
				WS:               tt.fields.WS,
				WSExternal:       tt.fields.WSExternal,
				WSUnsafe:         tt.fields.WSUnsafe,
				WSUnsafeExternal: tt.fields.WSUnsafeExternal,
			}
			got := r.isRPCEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_isWSEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
		Enabled          bool
		External         bool
		Unsafe           bool
		UnsafeExternal   bool
		Port             uint32
		Host             string
		Modules          []string
		WSPort           uint32
		WS               bool
		WSExternal       bool
		WSUnsafe         bool
		WSUnsafeExternal bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			want: false,
		},
		{
			name:   "ws true",
			fields: fields{WS: true},
			want:   true,
		},
		{
			name:   "ws external true",
			fields: fields{WSExternal: true},
			want:   true,
		},
		{
			name:   "ws unsafe true",
			fields: fields{WSUnsafe: true},
			want:   true,
		},
		{
			name:   "ws unsafe external true",
			fields: fields{WSUnsafeExternal: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RPCConfig{
				Enabled:          tt.fields.Enabled,
				External:         tt.fields.External,
				Unsafe:           tt.fields.Unsafe,
				UnsafeExternal:   tt.fields.UnsafeExternal,
				Port:             tt.fields.Port,
				Host:             tt.fields.Host,
				Modules:          tt.fields.Modules,
				WSPort:           tt.fields.WSPort,
				WS:               tt.fields.WS,
				WSExternal:       tt.fields.WSExternal,
				WSUnsafe:         tt.fields.WSUnsafe,
				WSUnsafeExternal: tt.fields.WSUnsafeExternal,
			}
			got := r.isWSEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_networkServiceEnabled(t *testing.T) {
	t.Parallel()

	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "dev config",
			args: args{cfg: DevConfig()},
			want: true,
		},
		{
			name: "empty config",
			args: args{cfg: &Config{}},
			want: false,
		},
		{
			name: "core roles 0",
			args: args{cfg: &Config{
				Core: CoreConfig{
					Roles: 0,
				},
			}},
			want: false,
		},
		{
			name: "core roles 1",
			args: args{cfg: &Config{
				Core: CoreConfig{
					Roles: 1,
				},
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := networkServiceEnabled(tt.args.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}
