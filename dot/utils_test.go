// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils/config"

	westend_dev "github.com/ChainSafe/gossamer/chain/westend-dev"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/internal/pprof"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGenesisFile returns a human-readable test genesis file using "westend-dev" human readable data
func newTestGenesisFile(t *testing.T, config *cfg.Config) (filename string) {
	t.Helper()

	fp := utils.GetWestendDevRawGenesisPath(t)
	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       config.Name,
		ID:         config.ID,
		Bootnodes:  config.Network.Bootnodes,
		ProtocolID: config.Network.ProtocolID,
		Genesis:    westendDevGenesis.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}

func TestCreateJSONRawFile(t *testing.T) {
	type args struct {
		bs *BuildSpec
		fp string
	}
	tests := []struct {
		name         string
		args         args
		expectedHash string
	}{
		{
			name: "working_example",
			args: args{
				bs: &BuildSpec{genesis: NewTestGenesis(t)},
				fp: filepath.Join(t.TempDir(), "/test.json"),
			},
			expectedHash: "f7f1b82c0ba16b20e36bfb462d7899af2c76728918f639f5c5ef0e91ff3e7077",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CreateJSONRawFile(tt.args.bs, tt.args.fp)

			b, err := os.ReadFile(tt.args.fp)
			require.NoError(t, err)
			digest := sha256.Sum256(b)
			hexDigest := fmt.Sprintf("%x", digest)
			require.Equal(t, tt.expectedHash, hexDigest)
		})
	}
}

func TestNewTestConfig(t *testing.T) {
	basePath := t.TempDir()
	incBasePath := basePath[:len(basePath)-1] + "2"
	type args struct {
		t *testing.T
	}
	tests := []struct {
		name string
		args args
		want *Config
	}{
		{
			name: "working_example",
			args: args{t: t},
			want: &Config{
				Global: GlobalConfig{
					Name:               "Westend",
					ID:                 "westend_dev",
					BasePath:           incBasePath,
					LogLvl:             3,
					PrometheusExternal: false,
					PrometheusPort:     0,
					NoTelemetry:        true,
					TelemetryURLs:      nil,
					RetainBlocks:       0,
					Pruning:            "",
				},
				Log: LogConfig{
					CoreLvl:           3,
					DigestLvl:         3,
					SyncLvl:           3,
					NetworkLvl:        3,
					RPCLvl:            3,
					StateLvl:          3,
					RuntimeLvl:        3,
					BlockProducerLvl:  3,
					FinalityGadgetLvl: 3,
				},
				Account: AccountConfig{Key: config.AliceKey},
				Init:    InitConfig{Genesis: "./chain/westend-dev/westend-dev-spec-raw.json"},
				Core: CoreConfig{
					Roles:            4,
					BabeAuthority:    true,
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
					PersistentPeers:   nil,
					DiscoveryInterval: 10000000000,
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
					WSPort:           8546,
					WS:               true,
					WSExternal:       false,
					WSUnsafe:         false,
					WSUnsafeExternal: false,
				},
				System: types.SystemInfo{},
				State:  StateConfig{},
				Pprof: PprofConfig{
					Settings: pprof.Settings{
						ListeningAddress: "localhost:6060",
						BlockProfileRate: 0,
						MutexProfileRate: 0,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWestendDevConfig(tt.args.t)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewTestGenesisFile(t *testing.T) {
	type args struct {
		t      *testing.T
		config *cfg.Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working_example",
			args: args{
				t:      t,
				config: westend_dev.DefaultConfig(),
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newTestGenesisFile(tt.args.t, tt.args.config)
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
			name: "working_example",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomNodeName()
			assert.Regexp(t, "[a-z]*-[a-z]*-[0-9]*", got)
		})
	}
}

// NewWestendDevConfig returns a new test configuration
func NewWestendDevConfig(t *testing.T) *Config {
	dir := t.TempDir()
	defaultConfig := WestendDevConfig()

	config := &Config{
		Global: GlobalConfig{
			Name:        defaultConfig.Global.Name,
			ID:          defaultConfig.Global.ID,
			BasePath:    dir,
			LogLvl:      defaultConfig.Global.LogLvl,
			NoTelemetry: true,
		},
		Init: InitConfig{
			Genesis: defaultConfig.Init.Genesis,
		},
		Log:     defaultConfig.Log,
		Account: defaultConfig.Account,
		Core:    defaultConfig.Core,
		Network: defaultConfig.Network,
		RPC:     defaultConfig.RPC,
		System:  defaultConfig.System,
		State:   defaultConfig.State,
		Pprof:   defaultConfig.Pprof,
	}

	return config
}

// NewTestGenesis returns a test genesis instance using "westend-dev" raw data
func NewTestGenesis(t *testing.T) *genesis.Genesis {
	fp := utils.GetWestendDevRawGenesisPath(t)

	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	return &genesis.Genesis{
		Name:       "test",
		ID:         "test",
		Bootnodes:  []string(nil),
		ProtocolID: "/gossamer/test/0",
		Genesis:    westendDevGenesis.GenesisFields(),
	}
}
