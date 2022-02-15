// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfigWithFile returns a new test configuration and a temporary configuration file
func newTestConfigWithFile(t *testing.T) (*Config, *os.File) {
	cfg := NewTestConfig(t)

	configPath := filepath.Join(cfg.Global.BasePath, "config.toml")
	err := os.WriteFile(configPath, nil, os.ModePerm)
	require.NoError(t, err)

	cfgFile := exportConfig(cfg, configPath)
	return cfg, cfgFile
}

// newTestGenesisFile returns a human-readable test genesis file using "gssmr" human readable data
func newTestGenesisFile(t *testing.T, cfg *Config) (filename string) {
	fp := utils.GetGssmrGenesisPath()

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
			got := exportConfig(tt.args.cfg, tt.args.fp)
			if tt.want != nil {
				require.Equal(t, tt.args.fp, got.Name())

				content, err := ioutil.ReadFile(got.Name())
				require.NoError(t, err)
				require.Equal(t, tt.wantedContent, string(content))
			}
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
		want          *os.File
		wantedContent string
	}{
		{
			name: "working example",
			args: args{
				cfg: &ctoml.Config{},
				fp:  filepath,
			},
			want: &os.File{},
			wantedContent: `[core]
babe-authority = false
grandpa-authority = false
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExportTomlConfig(tt.args.cfg, tt.args.fp)
			if tt.want != nil {
				require.Equal(t, tt.args.fp, got.Name())

				content, err := ioutil.ReadFile(got.Name())
				require.NoError(t, err)
				require.Equal(t, tt.wantedContent, string(content))
			}
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

func TestNewTestConfigWithFile(t *testing.T) {
	basePath := t.TempDir()
	incBasePath := basePath[:len(basePath)-1] + "2"
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
			want1: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := newTestConfigWithFile(tt.args.t)

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
			got := writeConfig(tt.args.data, tt.args.fp)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
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
	fp := utils.GetGssmrGenesisRawPath()

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

	fp := utils.GetGssmrGenesisRawPath()

	gssmrGen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.Nil(t, err)

	gen := &genesis.Genesis{
		Name:       cfg.Global.Name,
		ID:         cfg.Global.ID,
		Bootnodes:  cfg.Network.Bootnodes,
		ProtocolID: cfg.Network.ProtocolID,
		Genesis:    gssmrGen.GenesisFields(),
	}

	b, err := json.Marshal(gen)
	require.Nil(t, err)

	err = os.WriteFile(filename, b, os.ModePerm)
	require.NoError(t, err)

	return filename
}

// newTestGenesisAndRuntime create a new test runtime and a new test genesis
// file with the test runtime stored in raw data and returns the genesis file
func newTestGenesisAndRuntime(t *testing.T) (filename string) {
	_ = wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)
	runtimeFilePath := runtime.GetAbsolutePath(runtime.NODE_RUNTIME_FP)

	runtimeData, err := os.ReadFile(filepath.Clean(runtimeFilePath))
	require.Nil(t, err)

	gen := NewTestGenesis(t)
	hex := hex.EncodeToString(runtimeData)

	gen.Genesis.Raw = map[string]map[string]string{}
	if gen.Genesis.Raw["top"] == nil {
		gen.Genesis.Raw["top"] = make(map[string]string)
	}
	gen.Genesis.Raw["top"]["0x3a636f6465"] = "0x" + hex
	gen.Genesis.Raw["top"]["0xcf722c0832b5231d35e29f319ff27389f5032bfc7bfc3ba5ed7839f2042fb99f"] = "0x0000000000000001"

	genData, err := json.Marshal(gen)
	require.NoError(t, err)

	filename = filepath.Join(t.TempDir(), "genesis.json")
	err = os.WriteFile(filename, genData, os.ModePerm)
	require.NoError(t, err)

	return filename
}
