// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"fmt"
	"strings"
	"time"

	"github.com/ChainSafe/gossamer/chain/dev"
	"github.com/ChainSafe/gossamer/chain/gssmr"
	"github.com/ChainSafe/gossamer/chain/kusama"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/pprof"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
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
	Pruning        bool
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

// GssmrConfig returns a new test configuration using the provided basepath
func GssmrConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           gssmr.DefaultName,
			ID:             gssmr.DefaultID,
			BasePath:       gssmr.DefaultBasePath,
			LogLvl:         gssmr.DefaultLvl,
			MetricsAddress: gssmr.DefaultMetricsAddress,
			RetainBlocks:   gssmr.DefaultRetainBlocks,
			Pruning:        gssmr.DefaultPruningEnabled,
			TelemetryURLs:  gssmr.DefaultTelemetryURLs,
		},
		Log: LogConfig{
			CoreLvl:           gssmr.DefaultLvl,
			DigestLvl:         gssmr.DefaultLvl,
			SyncLvl:           gssmr.DefaultLvl,
			NetworkLvl:        gssmr.DefaultLvl,
			RPCLvl:            gssmr.DefaultLvl,
			StateLvl:          gssmr.DefaultLvl,
			RuntimeLvl:        gssmr.DefaultLvl,
			BlockProducerLvl:  gssmr.DefaultLvl,
			FinalityGadgetLvl: gssmr.DefaultLvl,
		},
		Init: InitConfig{
			Genesis: gssmr.DefaultGenesis,
		},
		Account: AccountConfig{
			Key:    gssmr.DefaultKey,
			Unlock: gssmr.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:            gssmr.DefaultRoles,
			BabeAuthority:    gssmr.DefaultBabeAuthority,
			GrandpaAuthority: gssmr.DefaultGrandpaAuthority,
			WasmInterpreter:  gssmr.DefaultWasmInterpreter,
			GrandpaInterval:  gssmr.DefaultGrandpaInterval,
		},
		Network: NetworkConfig{
			Port:              gssmr.DefaultNetworkPort,
			Bootnodes:         gssmr.DefaultNetworkBootnodes,
			NoBootstrap:       gssmr.DefaultNoBootstrap,
			NoMDNS:            gssmr.DefaultNoMDNS,
			DiscoveryInterval: gssmr.DefaultDiscoveryInterval,
			MinPeers:          gssmr.DefaultMinPeers,
			MaxPeers:          gssmr.DefaultMaxPeers,
		},
		RPC: RPCConfig{
			Port:    gssmr.DefaultRPCHTTPPort,
			Host:    gssmr.DefaultRPCHTTPHost,
			Modules: gssmr.DefaultRPCModules,
			WSPort:  gssmr.DefaultRPCWSPort,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: gssmr.DefaultPprofListeningAddress,
				BlockProfileRate: gssmr.DefaultPprofBlockRate,
				MutexProfileRate: gssmr.DefaultPprofMutexRate,
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
			RetainBlocks:   gssmr.DefaultRetainBlocks,
			Pruning:        gssmr.DefaultPruningEnabled,
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
			RetainBlocks:   gssmr.DefaultRetainBlocks,
			Pruning:        gssmr.DefaultPruningEnabled,
			MetricsAddress: gssmr.DefaultMetricsAddress,
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

// DevConfig returns the configuration for a development chain
func DevConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:           dev.DefaultName,
			ID:             dev.DefaultID,
			BasePath:       dev.DefaultBasePath,
			LogLvl:         dev.DefaultLvl,
			MetricsAddress: dev.DefaultMetricsAddress,
			RetainBlocks:   dev.DefaultRetainBlocks,
			Pruning:        dev.DefaultPruningEnabled,
			TelemetryURLs:  dev.DefaultTelemetryURLs,
		},
		Log: LogConfig{
			CoreLvl:           dev.DefaultLvl,
			DigestLvl:         dev.DefaultLvl,
			SyncLvl:           dev.DefaultLvl,
			NetworkLvl:        dev.DefaultLvl,
			RPCLvl:            dev.DefaultLvl,
			StateLvl:          dev.DefaultLvl,
			RuntimeLvl:        dev.DefaultLvl,
			BlockProducerLvl:  dev.DefaultLvl,
			FinalityGadgetLvl: dev.DefaultLvl,
		},
		Init: InitConfig{
			Genesis: dev.DefaultGenesis,
		},
		Account: AccountConfig{
			Key:    dev.DefaultKey,
			Unlock: dev.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:            dev.DefaultRoles,
			BabeAuthority:    dev.DefaultBabeAuthority,
			GrandpaAuthority: dev.DefaultGrandpaAuthority,
			WasmInterpreter:  dev.DefaultWasmInterpreter,
			BABELead:         dev.DefaultBabeAuthority,
		},
		Network: NetworkConfig{
			Port:        dev.DefaultNetworkPort,
			Bootnodes:   dev.DefaultNetworkBootnodes,
			NoBootstrap: dev.DefaultNoBootstrap,
			NoMDNS:      dev.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    dev.DefaultRPCHTTPPort,
			Host:    dev.DefaultRPCHTTPHost,
			Modules: dev.DefaultRPCModules,
			WSPort:  dev.DefaultRPCWSPort,
			Enabled: dev.DefaultRPCEnabled,
			WS:      dev.DefaultWSEnabled,
		},
		Pprof: PprofConfig{
			Settings: pprof.Settings{
				ListeningAddress: dev.DefaultPprofListeningAddress,
				BlockProfileRate: dev.DefaultPprofBlockRate,
				MutexProfileRate: dev.DefaultPprofMutexRate,
			},
		},
	}
}
