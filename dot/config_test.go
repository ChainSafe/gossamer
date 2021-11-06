// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
)

func TestConfig_String(t *testing.T) {
	type fields struct {
		Global  GlobalConfig
		Log     LogConfig
		Init    InitConfig
		Account AccountConfig
		Core    CoreConfig
		Network NetworkConfig
		RPC     RPCConfig
		System  types.SystemInfo
		State   StateConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Global:  tt.fields.Global,
				Log:     tt.fields.Log,
				Init:    tt.fields.Init,
				Account: tt.fields.Account,
				Core:    tt.fields.Core,
				Network: tt.fields.Network,
				RPC:     tt.fields.RPC,
				System:  tt.fields.System,
				State:   tt.fields.State,
			}
			if got := c.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DevConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DevConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGssmrConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GssmrConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GssmrConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKusamaConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KusamaConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KusamaConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolkadotConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PolkadotConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PolkadotConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRPCConfig_isRPCEnabled(t *testing.T) {
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
		// TODO: Add test cases.
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
			if got := r.isRPCEnabled(); got != tt.want {
				t.Errorf("isRPCEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRPCConfig_isWSEnabled(t *testing.T) {
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
		// TODO: Add test cases.
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
			if got := r.isWSEnabled(); got != tt.want {
				t.Errorf("isWSEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_networkServiceEnabled(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkServiceEnabled(tt.args.cfg); got != tt.want {
				t.Errorf("networkServiceEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
