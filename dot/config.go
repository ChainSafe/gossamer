// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"fmt"
	"strings"
	"time"

	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/chain/westend"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/chain/dev"
	"github.com/ChainSafe/gossamer/internal/chain/gssmr"
	"github.com/ChainSafe/gossamer/internal/chain/kusama"
	"github.com/ChainSafe/gossamer/internal/chain/polkadot"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/pprof"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

// TODO: update config to have toml rules and perhaps un-export some fields, since we don't want to expose all
// the internal config options, also type conversions might be needed from toml -> internal types (#1848)

// Config is a collection of configurations throughout the system
type Config struct {
	Global  GlobalConfig
	Log     LogConfig
	Init    InitConfig
	Account AccountConfig
	Core    CoreConfig
	Network NetworkConfig
	RPC     RPCConfig
	System  types.SystemInfo
	State   StateConfig
	Pprof   PprofConfig
}

// GlobalConfig is used for every node command
type GlobalConfig struct {
	Name           string
	ID             string
	BasePath       string
	LogLvl         log.Level
	PublishMetrics bool
	MetricsAddress string
	NoTelemetry    bool
	TelemetryURLs  []genesis.TelemetryEndpoint
	RetainBlocks   uint32
	Pruning        pruner.Mode
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
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

func (l LogConfig) String() string {
	entries := []string{
		fmt.Sprintf("core: %s", l.CoreLvl),
		fmt.Sprintf("digest: %s", l.DigestLvl),
		fmt.Sprintf("sync: %s", l.SyncLvl),
		fmt.Sprintf("network: %s", l.NetworkLvl),
		fmt.Sprintf("rpc: %s", l.RPCLvl),
		fmt.Sprintf("state: %s", l.StateLvl),
		fmt.Sprintf("runtime: %s", l.RuntimeLvl),
		fmt.Sprintf("block producer: %s", l.BlockProducerLvl),
		fmt.Sprintf("finality gadget: %s", l.FinalityGadgetLvl),
	}
	return strings.Join(entries, ", ")
}

// InitConfig is the configuration for the node initialization
type InitConfig struct {
	Genesis string
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string
	Unlock string // TODO: change to []int (#1849)
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port              uint16
	Bootnodes         []string
	ProtocolID        string
	NoBootstrap       bool
	NoMDNS            bool
	MinPeers          int
	MaxPeers          int
	PersistentPeers   []string
	DiscoveryInterval time.Duration
	PublicIP          string
	PublicDNS         string
	NodeKey           string
	ListenAddress     string
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Roles            common.Roles
	BabeAuthority    bool
	BABELead         bool
	GrandpaAuthority bool
	WasmInterpreter  string
	GrandpaInterval  time.Duration
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
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

func (r *RPCConfig) isRPCEnabled() bool {
	return r.Enabled || r.External || r.Unsafe || r.UnsafeExternal
}

func (r *RPCConfig) isWSEnabled() bool {
	return r.WS || r.WSExternal || r.WSUnsafe || r.WSUnsafeExternal
}

// Strings returns the configuration in the format
// field1=value1 field2=value2.
func (r *RPCConfig) String() string {
	return "" +
		"enabled=" + fmt.Sprint(r.Enabled) + " " +
		"external=" + fmt.Sprint(r.External) + " " +
		"unsafe=" + fmt.Sprint(r.Unsafe) + " " +
		"unsafeexternal=" + fmt.Sprint(r.UnsafeExternal) + " " +
		"port=" + fmt.Sprint(r.Port) + " " +
		"host=" + r.Host + " " +
		"modules=" + strings.Join(r.Modules, ",") + " " +
		"wsport=" + fmt.Sprint(r.WSPort) + " " +
		"ws=" + fmt.Sprint(r.WS) + " " +
		"wsexternal=" + fmt.Sprint(r.WSExternal) + " " +
		"wsunsafe=" + fmt.Sprint(r.WSUnsafe) + " " +
		"wsunsafeexternal=" + fmt.Sprint(r.WSUnsafeExternal)
}

// StateConfig is the config for the State service
type StateConfig struct {
	Rewind uint
}

func (s *StateConfig) String() string {
	return "rewind " + fmt.Sprint(s.Rewind)
}

// networkServiceEnabled returns true if the network service is enabled
func networkServiceEnabled(cfg *Config) bool {
	return cfg.Core.Roles != common.NoNetworkRole
}

// PprofConfig is the configuration for the pprof HTTP server.
type PprofConfig struct {
	Enabled  bool
	Settings pprof.Settings
}

func (p PprofConfig) String() string {
	if !p.Enabled {
		return "disabled"
	}

	return p.Settings.String()
}

// WestendDevConfig returns a westend node configuration
func WestendDevConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           "Westend",
			ID:             "westend_dev",
			BasePath:       "~/.gossamer/westend-dev",
			LogLvl:         log.Info,
			MetricsAddress: ":9876",
			RetainBlocks:   512,
			Pruning:        pruner.Archive,
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
			Genesis: "./chain/westend-dev/westend-dev-spec-raw.json",
		},
		Account: AccountConfig{
			Key:    "alice",
			Unlock: "",
		},
		Core: CoreConfig{
			Roles:            common.AuthorityRole,
			WasmInterpreter:  wasmer.Name,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  time.Second,
		},
		Network: NetworkConfig{
			Port:              7001,
			Bootnodes:         []string(nil),
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
		},
		RPC: RPCConfig{
			WS:      true,
			Enabled: true,
			Port:    8545,
			Host:    "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
			WSPort: 8546,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: "localhost:6060",
				BlockProfileRate: 0,
				MutexProfileRate: 0,
			},
		},
	}
}

// KusamaConfig returns a kusama node configuration
func KusamaConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           kusama.DefaultName,
			ID:             kusama.DefaultID,
			BasePath:       kusama.DefaultBasePath,
			LogLvl:         kusama.DefaultLvl,
			MetricsAddress: kusama.DefaultMetricsAddress,
			RetainBlocks:   kusama.DefaultRetainBlocks,
			Pruning:        pruner.Mode(kusama.DefaultPruningMode),
			TelemetryURLs:  kusama.DefaultTelemetryURLs,
		},
		Log: LogConfig{
			CoreLvl:           kusama.DefaultLvl,
			DigestLvl:         kusama.DefaultLvl,
			SyncLvl:           kusama.DefaultLvl,
			NetworkLvl:        kusama.DefaultLvl,
			RPCLvl:            kusama.DefaultLvl,
			StateLvl:          kusama.DefaultLvl,
			RuntimeLvl:        kusama.DefaultLvl,
			BlockProducerLvl:  kusama.DefaultLvl,
			FinalityGadgetLvl: kusama.DefaultLvl,
		},
		Init: InitConfig{
			Genesis: kusama.DefaultGenesis,
		},
		Account: AccountConfig{
			Key:    kusama.DefaultKey,
			Unlock: kusama.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:           kusama.DefaultRoles,
			WasmInterpreter: kusama.DefaultWasmInterpreter,
		},
		Network: NetworkConfig{
			Port:        kusama.DefaultNetworkPort,
			Bootnodes:   kusama.DefaultNetworkBootnodes,
			NoBootstrap: kusama.DefaultNoBootstrap,
			NoMDNS:      kusama.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    kusama.DefaultRPCHTTPPort,
			Host:    kusama.DefaultRPCHTTPHost,
			Modules: kusama.DefaultRPCModules,
			WSPort:  kusama.DefaultRPCWSPort,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: kusama.DefaultPprofListeningAddress,
				BlockProfileRate: kusama.DefaultPprofBlockRate,
				MutexProfileRate: kusama.DefaultPprofMutexRate,
			},
		},
	}
}

// PolkadotConfig returns a "polkadot" node configuration
func PolkadotConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           polkadot.DefaultName,
			ID:             polkadot.DefaultID,
			BasePath:       polkadot.DefaultBasePath,
			LogLvl:         polkadot.DefaultLvl,
			RetainBlocks:   polkadot.DefaultRetainBlocks,
			Pruning:        pruner.Mode(polkadot.DefaultPruningMode),
			MetricsAddress: polkadot.DefaultMetricsAddress,
			TelemetryURLs:  polkadot.DefaultTelemetryURLs,
		},
		Log: LogConfig{
			CoreLvl:           polkadot.DefaultLvl,
			DigestLvl:         polkadot.DefaultLvl,
			SyncLvl:           polkadot.DefaultLvl,
			NetworkLvl:        polkadot.DefaultLvl,
			RPCLvl:            polkadot.DefaultLvl,
			StateLvl:          polkadot.DefaultLvl,
			RuntimeLvl:        polkadot.DefaultLvl,
			BlockProducerLvl:  polkadot.DefaultLvl,
			FinalityGadgetLvl: polkadot.DefaultLvl,
		},
		Init: InitConfig{
			Genesis: polkadot.DefaultGenesis,
		},
		Account: AccountConfig{
			Key:    polkadot.DefaultKey,
			Unlock: polkadot.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:           polkadot.DefaultRoles,
			WasmInterpreter: polkadot.DefaultWasmInterpreter,
		},
		Network: NetworkConfig{
			Port:        polkadot.DefaultNetworkPort,
			Bootnodes:   polkadot.DefaultNetworkBootnodes,
			NoBootstrap: polkadot.DefaultNoBootstrap,
			NoMDNS:      polkadot.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    polkadot.DefaultRPCHTTPPort,
			Host:    polkadot.DefaultRPCHTTPHost,
			Modules: polkadot.DefaultRPCModules,
			WSPort:  polkadot.DefaultRPCWSPort,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: polkadot.DefaultPprofListeningAddress,
				BlockProfileRate: polkadot.DefaultPprofBlockRate,
				MutexProfileRate: polkadot.DefaultPprofMutexRate,
			},
		},
	}
}

// WestendConfig returns a "westend" node configuration
func WestendConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           westend.DefaultName,
			ID:             westend.DefaultID,
			BasePath:       westend.DefaultBasePath,
			LogLvl:         westend.DefaultLvl,
			RetainBlocks:   westend.DefaultRetainBlocks,
			Pruning:        pruner.Mode(westend.DefaultPruningMode),
			MetricsAddress: westend.DefaultMetricsAddress,
			TelemetryURLs:  westend.DefaultTelemetryURLs,
		},
		Log: LogConfig{
			CoreLvl:           westend.DefaultLvl,
			DigestLvl:         westend.DefaultLvl,
			SyncLvl:           westend.DefaultLvl,
			NetworkLvl:        westend.DefaultLvl,
			RPCLvl:            westend.DefaultLvl,
			StateLvl:          westend.DefaultLvl,
			RuntimeLvl:        westend.DefaultLvl,
			BlockProducerLvl:  westend.DefaultLvl,
			FinalityGadgetLvl: westend.DefaultLvl,
		},
		Init: InitConfig{
			Genesis: westend.DefaultGenesis,
		},
		Account: AccountConfig{
			Key:    westend.DefaultKey,
			Unlock: westend.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:           westend.DefaultRoles,
			WasmInterpreter: westend.DefaultWasmInterpreter,
		},
		Network: NetworkConfig{
			Port:        westend.DefaultNetworkPort,
			Bootnodes:   westend.DefaultNetworkBootnodes,
			NoBootstrap: westend.DefaultNoBootstrap,
			NoMDNS:      westend.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    westend.DefaultRPCHTTPPort,
			Host:    westend.DefaultRPCHTTPHost,
			Modules: westend.DefaultRPCModules,
			WSPort:  westend.DefaultRPCWSPort,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: westend.DefaultPprofListeningAddress,
				BlockProfileRate: westend.DefaultPprofBlockRate,
				MutexProfileRate: westend.DefaultPprofMutexRate,
			},
		},
	}
}
