// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGenesisFile returns a human-readable test genesis file using "westend-dev" human readable data
func newTestGenesisFile(t *testing.T, cfg *Config) (filename string) {
	t.Helper()

	fp := utils.GetWestendDevRawGenesisPath(t)
	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    westendDevGenesis.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
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
					Name:           "Westend",
					ID:             "westend_dev",
					BasePath:       incBasePath,
					LogLvl:         3,
					PublishMetrics: false,
					MetricsAddress: "",
					NoTelemetry:    true,
					TelemetryURLs:  nil,
					RetainBlocks:   0,
					Pruning:        "",
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
				Account: AccountConfig{Key: "alice"},
				Init:    InitConfig{Genesis: "./chain/westend-dev/westend-dev-spec-raw.json"},
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
		t   *testing.T
		cfg *Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working_example",
			args: args{
				t:   t,
				cfg: &Config{},
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newTestGenesisFile(tt.args.t, tt.args.cfg)
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

	cfg := &Config{
		Global: GlobalConfig{
			Name:        WestendDevConfig().Global.Name,
			ID:          WestendDevConfig().Global.ID,
			BasePath:    dir,
			LogLvl:      log.Info,
			NoTelemetry: true,
		},
		Log:     WestendDevConfig().Log,
		Init:    WestendDevConfig().Init,
		Account: WestendDevConfig().Account,
		Core:    WestendDevConfig().Core,
		Network: WestendDevConfig().Network,
		RPC:     WestendDevConfig().RPC,
	}

	return cfg
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
