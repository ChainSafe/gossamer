// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/pprof"
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
					Name:         "Gossamer",
					ID:           "dev",
					BasePath:     "~/.gossamer/dev",
					LogLvl:       log.Info,
					MetricsPort:  9876,
					RetainBlocks: 512,
					Pruning:      "archive",
				},
				Log: LogConfig{
					CoreLvl:           log.Info,
					DigestLvl:         log.Info,
					SyncLvl:           log.Info,
					NetworkLvl:        log.Info,
					RPCLvl:            log.Info,
					StateLvl:          log.Info,
					RuntimeLvl:        log.Info,
					BlockProducerLvl:  log.Info,
					FinalityGadgetLvl: log.Info,
				},
				Init: InitConfig{
					Genesis: "./chain/dev/genesis-spec.json",
				},
				Account: AccountConfig{
					Key: "alice",
				},
				Core: CoreConfig{
					Roles:            byte(4),
					BabeAuthority:    true,
					BABELead:         true,
					GrandpaAuthority: true,
					WasmInterpreter:  "wasmer",
					GrandpaInterval:  0,
				},
				Network: NetworkConfig{
					Port: 7001,
				},
				RPC: RPCConfig{
					Enabled:        true,
					External:       false,
					Unsafe:         false,
					UnsafeExternal: false,
					Port:           8545,
					Host:           "localhost",
					Modules: []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain",
						"childstate", "syncstate", "payment"},
					WSPort: 8546,
					WS:     true,
				},
				Pprof: PprofConfig{
					Enabled: true,
					Settings: pprof.Settings{
						ListeningAddress: "localhost:6060",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DevConfig()
			assert.Equal(t, tt.want, got)
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
					Name:         "Gossamer",
					ID:           "gssmr",
					BasePath:     "~/.gossamer/gssmr",
					LogLvl:       log.Info,
					MetricsPort:  9876,
					RetainBlocks: 512,
					Pruning:      "archive",
				},
				Log: LogConfig{
					CoreLvl:           log.Info,
					DigestLvl:         log.Info,
					SyncLvl:           log.Info,
					NetworkLvl:        log.Info,
					RPCLvl:            log.Info,
					StateLvl:          log.Info,
					RuntimeLvl:        log.Info,
					BlockProducerLvl:  log.Info,
					FinalityGadgetLvl: log.Info,
				},
				Init: InitConfig{
					Genesis: "./chain/gssmr/genesis-spec.json",
				},
				Account: AccountConfig{},
				Core: CoreConfig{
					Roles:            byte(4),
					BabeAuthority:    true,
					GrandpaAuthority: true,
					WasmInterpreter:  "wasmer",
					GrandpaInterval:  time.Second,
				},
				Network: NetworkConfig{
					Port:              7001,
					MinPeers:          1,
					MaxPeers:          50,
					DiscoveryInterval: time.Second * 10,
				},
				RPC: RPCConfig{
					Port: 8545,
					Host: "localhost",
					Modules: []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain",
						"childstate", "syncstate", "payment"},
					WSPort:           8546,
					WS:               false,
					WSExternal:       false,
					WSUnsafe:         false,
					WSUnsafeExternal: false,
				},
				Pprof: PprofConfig{
					Enabled: true,
					Settings: pprof.Settings{
						ListeningAddress: "localhost:6060",
						BlockProfileRate: 0,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GssmrConfig()
			assert.Equal(t, tt.want, got)
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
					Name:         "Kusama",
					ID:           "ksmcc3",
					BasePath:     "~/.gossamer/kusama",
					LogLvl:       log.Info,
					MetricsPort:  9876,
					RetainBlocks: 512,
					Pruning:      "archive",
				},
				Log: LogConfig{
					CoreLvl:           log.Info,
					DigestLvl:         log.Info,
					SyncLvl:           log.Info,
					NetworkLvl:        log.Info,
					RPCLvl:            log.Info,
					StateLvl:          log.Info,
					RuntimeLvl:        log.Info,
					BlockProducerLvl:  log.Info,
					FinalityGadgetLvl: log.Info,
				},
				Init: InitConfig{
					Genesis: "./chain/kusama/genesis.json",
				},
				Account: AccountConfig{},
				Core: CoreConfig{
					Roles:           byte(1),
					WasmInterpreter: "wasmer",
					GrandpaInterval: 0,
				},
				Network: NetworkConfig{
					Port:              7001,
					Bootnodes:         nil,
					ProtocolID:        "",
					NoBootstrap:       false,
					NoMDNS:            false,
					MinPeers:          0,
					MaxPeers:          0,
					PersistentPeers:   nil,
					DiscoveryInterval: 0,
					PublicIP:          "",
					PublicDNS:         "",
				},
				RPC: RPCConfig{
					Port: 8545,
					Host: "localhost",
					Modules: []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain",
						"childstate", "syncstate", "payment"},
					WSPort: 8546,
				},
				Pprof: PprofConfig{
					Settings: pprof.Settings{
						ListeningAddress: "localhost:6060",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := KusamaConfig()
			assert.Equal(t, tt.want, got)
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
					Name:         "Polkadot",
					ID:           "polkadot",
					BasePath:     "~/.gossamer/polkadot",
					LogLvl:       log.Info,
					MetricsPort:  9876,
					RetainBlocks: 512,
					Pruning:      "archive",
				},
				Log: LogConfig{
					CoreLvl:           log.Info,
					DigestLvl:         log.Info,
					SyncLvl:           log.Info,
					NetworkLvl:        log.Info,
					RPCLvl:            log.Info,
					StateLvl:          log.Info,
					RuntimeLvl:        log.Info,
					BlockProducerLvl:  log.Info,
					FinalityGadgetLvl: log.Info,
				},
				Init: InitConfig{Genesis: "./chain/polkadot/genesis.json"},
				Core: CoreConfig{
					Roles:           byte(1),
					WasmInterpreter: "wasmer",
				},
				Network: NetworkConfig{
					Port: 7001,
				},
				RPC: RPCConfig{
					Port: 8545,
					Host: "localhost",
					Modules: []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain",
						"childstate", "syncstate", "payment"},
					WSPort: 8546,
				},
				Pprof: PprofConfig{
					Settings: pprof.Settings{
						ListeningAddress: "localhost:6060",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := PolkadotConfig()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_isRPCEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
		Enabled        bool
		External       bool
		Unsafe         bool
		UnsafeExternal bool
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &RPCConfig{
				Enabled:        tt.fields.Enabled,
				External:       tt.fields.External,
				Unsafe:         tt.fields.Unsafe,
				UnsafeExternal: tt.fields.UnsafeExternal,
			}
			got := r.isRPCEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_isWSEnabled(t *testing.T) {
	t.Parallel()

	type fields struct {
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &RPCConfig{
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := networkServiceEnabled(tt.args.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRPCConfig_String(t *testing.T) {
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
		want   string
	}{
		{
			name:   "default base case",
			fields: fields{},
			want: "enabled=false external=false unsafe=false unsafeexternal=false port=0 host= modules= wsport=0 ws" +
				"=false wsexternal=false wsunsafe=false wsunsafeexternal=false",
		},
		{
			name: "fields changed",
			fields: fields{
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
			assert.Equalf(t, tt.want, r.String(), "String()")
		})
	}
}

func TestLogConfig_String(t *testing.T) {
	type fields struct {
		CoreLvl           log.Level
		DigestLvl         log.Level
		SyncLvl           log.Level
		NetworkLvl        log.Level
		RPCLvl            log.Level
		StateLvl          log.Level
		RuntimeLvl        log.Level
		BlockProducerLvl  log.Level
		FinalityGadgetLvl log.Level
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "default case",
			fields: fields{},
			want: "core: CRIT, digest: CRIT, sync: CRIT, network: CRIT, rpc: CRIT, state: CRIT, runtime: CRIT, " +
				"block producer: CRIT, finality gadget: CRIT",
		},
		{
			name: "change fields case",
			fields: fields{
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
			want: "core: DBUG, digest: INFO, sync: WARN, network: EROR, rpc: CRIT, state: DBUG, runtime: INFO, " +
				"block producer: WARN, finality gadget: EROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LogConfig{
				CoreLvl:           tt.fields.CoreLvl,
				DigestLvl:         tt.fields.DigestLvl,
				SyncLvl:           tt.fields.SyncLvl,
				NetworkLvl:        tt.fields.NetworkLvl,
				RPCLvl:            tt.fields.RPCLvl,
				StateLvl:          tt.fields.StateLvl,
				RuntimeLvl:        tt.fields.RuntimeLvl,
				BlockProducerLvl:  tt.fields.BlockProducerLvl,
				FinalityGadgetLvl: tt.fields.FinalityGadgetLvl,
			}
			assert.Equalf(t, tt.want, l.String(), "String()")
		})
	}
}
