// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"os"
	"strings"
	"testing"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
)

func TestCreateJSONRawFile(t *testing.T) {
	type args struct {
		bs *BuildSpec
		fp string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				bs: &BuildSpec{genesis: NewTestGenesis(t)},
				fp: "test_data/test.json",
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateJSONRawFile(tt.args.bs, tt.args.fp)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestExportConfig(t *testing.T) {
	type args struct {
		cfg *Config
		fp  string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				cfg: &Config{},
				fp:  "test_data/test.json",
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExportConfig(tt.args.cfg, tt.args.fp)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestExportTomlConfig(t *testing.T) {
	type args struct {
		cfg *ctoml.Config
		fp  string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				cfg: &ctoml.Config{},
				fp:  "test_data/test.json",
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExportTomlConfig(tt.args.cfg, tt.args.fp)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewTestConfig(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		{
			name: "working example",
			args: args{t: t},
			want: &Config{
				Global: GlobalConfig{
					Name:           "Gossamer",
					ID:             "gssmr",
					BasePath:       "test_data/TestNewTestConfig",
					LogLvl:         3,
					PublishMetrics: false,
					MetricsPort:    0,
					NoTelemetry:    false,
					TelemetryURLs:  nil,
					RetainBlocks:   0,
					Pruning:        "",
				},
				Log: LogConfig{
					CoreLvl:           3,
					SyncLvl:           3,
					NetworkLvl:        3,
					RPCLvl:            3,
					StateLvl:          3,
					RuntimeLvl:        3,
					BlockProducerLvl:  3,
					FinalityGadgetLvl: 3,
				},
				Init: InitConfig{Genesis: "./chain/gssmr/genesis-spec.json"},
				Core: CoreConfig{
					Roles:            4,
					BabeAuthority:    true,
					BABELead:         false,
					GrandpaAuthority: true,
					WasmInterpreter:  "wasmer",
					GrandpaInterval:  1000000000,
				},
				Network: NetworkConfig{
					Port:              7001,
					Bootnodes:         nil,
					ProtocolID:        "",
					NoBootstrap:       false,
					NoMDNS:            false,
					MinPeers:          1,
					MaxPeers:          0,
					PersistentPeers:   nil,
					DiscoveryInterval: 10000000000,
				},
				RPC: RPCConfig{
					Enabled:          false,
					External:         false,
					Unsafe:           false,
					UnsafeExternal:   false,
					Port:             8545,
					Host:             "localhost",
					Modules:          []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain", "childstate", "syncstate", "payment"},
					WSPort:           8546,
					WS:               false,
					WSExternal:       false,
					WSUnsafe:         false,
					WSUnsafeExternal: false,
				},
				System: types.SystemInfo{},
				State:  StateConfig{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestConfig(tt.args.t)
			if tt.want != nil {
				assert.Equal(t, tt.want, got)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewTestConfigWithFile(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name  string
		args  args
		want  *Config
		want1 *os.File
	}{
		{
			name: "working example",
			args: args{t: t},
			want: &Config{
				Global: GlobalConfig{
					Name:           "Gossamer",
					ID:             "gssmr",
					BasePath:       "test_data/TestNewTestConfigWithFile",
					LogLvl:         3,
					PublishMetrics: false,
					MetricsPort:    0,
					NoTelemetry:    false,
					TelemetryURLs:  nil,
					RetainBlocks:   0,
					Pruning:        "",
				},
				Log: LogConfig{
					CoreLvl:           3,
					SyncLvl:           3,
					NetworkLvl:        3,
					RPCLvl:            3,
					StateLvl:          3,
					RuntimeLvl:        3,
					BlockProducerLvl:  3,
					FinalityGadgetLvl: 3,
				},
				Init: InitConfig{Genesis: "./chain/gssmr/genesis-spec.json"},
				Core: CoreConfig{
					Roles:            4,
					BabeAuthority:    true,
					BABELead:         false,
					GrandpaAuthority: true,
					WasmInterpreter:  "wasmer",
					GrandpaInterval:  1000000000,
				},
				Network: NetworkConfig{
					Port:              7001,
					Bootnodes:         nil,
					ProtocolID:        "",
					NoBootstrap:       false,
					NoMDNS:            false,
					MinPeers:          1,
					MaxPeers:          0,
					PersistentPeers:   nil,
					DiscoveryInterval: 10000000000,
				},
				RPC: RPCConfig{
					Enabled:          false,
					External:         false,
					Unsafe:           false,
					UnsafeExternal:   false,
					Port:             8545,
					Host:             "localhost",
					Modules:          []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain", "childstate", "syncstate", "payment"},
					WSPort:           8546,
					WS:               false,
					WSExternal:       false,
					WSUnsafe:         false,
					WSUnsafeExternal: false,
				},
				System: types.SystemInfo{},
				State:  StateConfig{},
			},
			want1: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := NewTestConfigWithFile(tt.args.t)

			assert.Equal(t, tt.want, got)
			if tt.want1 != nil {
				assert.NotNil(t, got1)
			}
		})
	}
}

func TestNewTestGenesis(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want *genesis.Genesis
	}{
		{
			name: "working example",
			args: args{t: t},
			want: &genesis.Genesis{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestGenesis(tt.args.t)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewTestGenesisAndRuntime(t *testing.T) {
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "working example",
			args: args{t: t},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestGenesisAndRuntime(tt.args.t)
			assert.True(t, strings.HasPrefix(got, "test_data/TestNewTestGenesisAndRuntime/genesis"))
		})
	}
}

func TestNewTestGenesisFile(t *testing.T) {
	type args struct {
		t   *testing.T
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				t:   t,
				cfg: &Config{},
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestGenesisFile(tt.args.t, tt.args.cfg)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewTestGenesisRawFile(t *testing.T) {
	type args struct {
		t   *testing.T
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				t:   t,
				cfg: &Config{},
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestGenesisRawFile(tt.args.t, tt.args.cfg)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestRandomNodeName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "working example",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomNodeName()
			assert.Greater(t, len(got), 3)
		})
	}
}

func TestWriteConfig(t *testing.T) {
	type args struct {
		data []byte
		fp   string
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working example",
			args: args{
				data: nil,
				fp:   "test_data/test.json",
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WriteConfig(tt.args.data, tt.args.fp)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}
