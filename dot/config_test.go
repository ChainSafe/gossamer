// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"testing"

	"github.com/ChainSafe/gossamer/chain/kusama"
	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestRPCConfig_isRPCEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rpcConfig *RPCConfig
		want      bool
	}{
		{
			name:      "default",
			rpcConfig: &RPCConfig{},
			want:      false,
		},
		{
			name:      "enabled true",
			rpcConfig: &RPCConfig{Enabled: true},
			want:      true,
		},
		{
			name:      "external true",
			rpcConfig: &RPCConfig{External: true},
			want:      true,
		},
		{
			name:      "unsafe true",
			rpcConfig: &RPCConfig{Unsafe: true},
			want:      true,
		},
		{
			name:      "unsafe external true",
			rpcConfig: &RPCConfig{UnsafeExternal: true},
			want:      true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.rpcConfig.isRPCEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_isWSEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rpcConfig *RPCConfig
		want      bool
	}{
		{
			name:      "default",
			rpcConfig: &RPCConfig{},
			want:      false,
		},
		{
			name:      "ws true",
			rpcConfig: &RPCConfig{WS: true},
			want:      true,
		},
		{
			name:      "ws external true",
			rpcConfig: &RPCConfig{WSExternal: true},
			want:      true,
		},
		{
			name:      "ws unsafe true",
			rpcConfig: &RPCConfig{WSUnsafe: true},
			want:      true,
		},
		{
			name:      "ws unsafe external true",
			rpcConfig: &RPCConfig{WSUnsafeExternal: true},
			want:      true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.rpcConfig.isWSEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_networkServiceEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *cfg.Config
		want   bool
	}{
		{
			name:   "kusama_config",
			config: kusama.DefaultConfig(),
			want:   true,
		},
		{
			name: "empty_config",
			config: &cfg.Config{
				BaseConfig: cfg.BaseConfig{},
				Log:        &cfg.LogConfig{},
				Account:    &cfg.AccountConfig{},
				Core:       &cfg.CoreConfig{},
				Network:    &cfg.NetworkConfig{},
				State:      &cfg.StateConfig{},
				RPC:        &cfg.RPCConfig{},
				Pprof:      &cfg.PprofConfig{},
				System:     &cfg.SystemConfig{},
			},
			want: false,
		},
		{
			name: "core_roles_0",
			config: &cfg.Config{
				Core: &cfg.CoreConfig{
					Role: 0,
				},
			},
			want: false,
		},
		{
			name: "core_roles_1",
			config: &cfg.Config{
				Core: &cfg.CoreConfig{
					Role: 1,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := networkServiceEnabled(tt.config)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_String(t *testing.T) {
	tests := []struct {
		name      string
		rpcConfig RPCConfig
		want      string
	}{
		{
			name:      "default base case",
			rpcConfig: RPCConfig{},
			want: "enabled=false external=false unsafe=false unsafeexternal=false port=0 host= modules= wsport=0 ws" +
				"=false wsexternal=false wsunsafe=false wsunsafeexternal=false",
		},
		{
			name: "fields_changed",
			rpcConfig: RPCConfig{
				Enabled:          true,
				External:         true,
				Unsafe:           true,
				UnsafeExternal:   true,
				Port:             1234,
				Host:             "5678",
				Modules:          nil,
				WSPort:           2345,
				WS:               true,
				WSExternal:       true,
				WSUnsafe:         true,
				WSUnsafeExternal: true,
			},
			want: "enabled=true external=true unsafe=true unsafeexternal=true port=1234 host=5678 modules= wsport" +
				"=2345 ws=true wsexternal=true wsunsafe=true wsunsafeexternal=true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rpcConfig.String())
		})
	}
}

func TestLogConfig_String(t *testing.T) {
	tests := []struct {
		name      string
		logConfig LogConfig
		want      string
	}{
		{
			name:      "default case",
			logConfig: LogConfig{},
			want: "core: CRITICAL, digest: CRITICAL, sync: CRITICAL, network: CRITICAL, rpc: CRITICAL, " +
				"state: CRITICAL, runtime: CRITICAL, block producer: CRITICAL, finality gadget: CRITICAL",
		},
		{
			name: "change_fields_case",
			logConfig: LogConfig{
				CoreLvl:           log.Debug,
				DigestLvl:         log.Info,
				SyncLvl:           log.Warn,
				NetworkLvl:        log.Error,
				RPCLvl:            log.Critical,
				StateLvl:          log.Debug,
				RuntimeLvl:        log.Info,
				BlockProducerLvl:  log.Warn,
				FinalityGadgetLvl: log.Error,
			},
			want: "core: DEBUG, digest: INFO, sync: WARN, network: ERROR, rpc: CRITICAL, " +
				"state: DEBUG, runtime: INFO, block producer: WARN, finality gadget: ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.logConfig.String())
		})
	}
}
