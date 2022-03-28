// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGenesisFile returns a human-readable test genesis file using "gssmr" human readable data
func newTestGenesisFile(t *testing.T, cfg *Config) (filename string) {
	t.Helper()

	fp := utils.GetGssmrGenesisPathTest(t)

	gssmrGen, err := genesis.NewGenesisFromJSON(fp, 0)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    gssmrGen.GenesisFields(),
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
			name: "working example",
			args: args{
				bs: &BuildSpec{genesis: NewTestGenesis(t)},
				fp: filepath.Join(t.TempDir(), "/test.json"),
			},
			expectedHash: "23356cdb5d3537d39b735726707216c9e329c7b8a2c8a41b25da0f5f936b3caa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CreateJSONRawFile(tt.args.bs, tt.args.fp)

			b, err := ioutil.ReadFile(tt.args.fp)
			require.NoError(t, err)
			digest := sha256.Sum256(b)
			hexDigest := fmt.Sprintf("%x", digest)
			require.Equal(t, tt.expectedHash, hexDigest)
		})
	}
}

func TestExportConfig(t *testing.T) {
	filepath := t.TempDir() + "/test.json"
	type args struct {
		cfg *Config
		fp  string
	}
	tests := []struct {
		name          string
		args          args
		want          *os.File
		wantedContent string
	}{
		{
			name: "working example",
			args: args{
				cfg: &Config{},
				fp:  filepath,
			},
			want: &os.File{},
			wantedContent: `[global]
name = ""
id = ""
base_path = ""
log_lvl = 0
publish_metrics = false
metrics_port = 0
no_telemetry = false
telemetry_urls = []
retain_blocks = 0
pruning = ""

[log]
core_lvl = 0
digest_lvl = 0
sync_lvl = 0
network_lvl = 0
rpc_lvl = 0
state_lvl = 0
runtime_lvl = 0
block_producer_lvl = 0
finality_gadget_lvl = 0

[init]
genesis = ""

[account]
key = ""
unlock = ""

[core]
roles = 0
babe_authority = false
b_a_b_e_lead = false
grandpa_authority = false
wasm_interpreter = ""
grandpa_interval = 0

[network]
port = 0
bootnodes = []
protocol_id = ""
no_bootstrap = false
no_m_dns = false
min_peers = 0
max_peers = 0
persistent_peers = []
discovery_interval = 0
public_ip = ""
public_dns = ""

[rpc]
enabled = false
external = false
unsafe = false
unsafe_external = false
port = 0
host = ""
modules = []
w_s_port = 0
w_s = false
w_s_external = false
w_s_unsafe = false
w_s_unsafe_external = false

[system]
system_name = ""
system_version = ""

[state]
rewind = 0

[pprof]
enabled = false

[pprof.settings]
listening_address = ""
block_profile_rate = 0
mutex_profile_rate = 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exportConfig(tt.args.cfg, tt.args.fp)

			content, err := ioutil.ReadFile(tt.args.fp)
			require.NoError(t, err)
			require.Equal(t, tt.wantedContent, string(content))

		})
	}
}

func TestExportTomlConfig(t *testing.T) {
	filepath := t.TempDir() + "/test.json"
	type args struct {
		cfg *ctoml.Config
		fp  string
	}
	tests := []struct {
		name          string
		args          args
		wantedContent string
	}{
		{
			name: "working example",
			args: args{
				cfg: &ctoml.Config{},
				fp:  filepath,
			},
			wantedContent: `[core]
babe-authority = false
grandpa-authority = false
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExportTomlConfig(tt.args.cfg, tt.args.fp)

			content, err := ioutil.ReadFile(tt.args.fp)
			require.NoError(t, err)
			require.Equal(t, tt.wantedContent, string(content))

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
			name: "working example",
			args: args{t: t},
			want: &Config{
				Global: GlobalConfig{
					Name:           "Gossamer",
					ID:             "gssmr",
					BasePath:       incBasePath,
					LogLvl:         3,
					PublishMetrics: false,
					MetricsPort:    0,
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
					MaxPeers:          50,
					PersistentPeers:   nil,
					DiscoveryInterval: 10000000000,
				},
				RPC: RPCConfig{
					Enabled:        false,
					External:       false,
					Unsafe:         false,
					UnsafeExternal: false,
					Port:           8545,
					Host:           "localhost",
					Modules: []string{"system", "author", "chain", "state", "rpc", "grandpa", "offchain",
						"childstate", "syncstate", "payment"},
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
			name: "working example",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomNodeName()
			assert.Regexp(t, "[a-z]*-[a-z]*-[0-9]*", got)
		})
	}
}

// NewTestConfig returns a new test configuration using the provided basepath
func NewTestConfig(t *testing.T) *Config {
	dir := t.TempDir()

	cfg := &Config{
		Global: GlobalConfig{
			Name:        GssmrConfig().Global.Name,
			ID:          GssmrConfig().Global.ID,
			BasePath:    dir,
			LogLvl:      log.Info,
			NoTelemetry: true,
		},
		Log:     GssmrConfig().Log,
		Init:    GssmrConfig().Init,
		Account: GssmrConfig().Account,
		Core:    GssmrConfig().Core,
		Network: GssmrConfig().Network,
		RPC:     GssmrConfig().RPC,
	}

	return cfg
}

// NewTestGenesis returns a test genesis instance using "gssmr" raw data
func NewTestGenesis(t *testing.T) *genesis.Genesis {
	fp, err := utils.GetGssmrGenesisRawPath()
	require.NoError(t, err)

	gssmrGen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	return &genesis.Genesis{
		Name:       "test",
		ID:         "test",
		Bootnodes:  []string(nil),
		ProtocolID: "/gossamer/test/0",
		Genesis:    gssmrGen.GenesisFields(),
	}
}

// newTestGenesisRawFile returns a test genesis file using "gssmr" raw data
func newTestGenesisRawFile(t *testing.T, cfg *Config) (filename string) {
	filename = filepath.Join(t.TempDir(), "genesis.json")

	fp, err := utils.GetGssmrGenesisRawPath()
	require.NoError(t, err)

	gssmrGen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    gssmrGen.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.NoError(t, err)

	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}
