// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/os"
	wazero "github.com/ChainSafe/gossamer/lib/runtime/wazero"
)

const (
	// uint32Max is the maximum value of a uint32
	uint32Max = ^uint32(0)
	// defaultChainSpecFile is the default genesis file
	defaultChainSpecFile = "chain-spec-raw.json"
	// defaultBasePath is the default base path
	defaultBasePath = "~/.gossamer/gssmr"
	// DefaultLogLevel is the default log level
	DefaultLogLevel = "info"
	// DefaultPrometheusPort is the default prometheus port
	DefaultPrometheusPort = uint32(9876)
	// DefaultRetainBlocks is the default number of blocks to retain
	DefaultRetainBlocks = 512
	// DefaultPruning is the default pruning strategy
	DefaultPruning = pruner.Archive

	// defaultAccount is the default account key
	defaultAccount = "alice"

	// DefaultRole is the default node role
	DefaultRole = common.AuthorityRole
	// DefaultWasmInterpreter is the default wasm interpreter
	DefaultWasmInterpreter = wazero.Name

	// DefaultNetworkPort is the default network port
	DefaultNetworkPort = 7001
	// DefaultDiscoveryInterval is the default discovery interval
	DefaultDiscoveryInterval = 10 * time.Second
	// DefaultMinPeers is the default minimum number of peers
	DefaultMinPeers = 0
	// DefaultMaxPeers is the default maximum number of peers
	DefaultMaxPeers = 50

	// DefaultRPCPort is the default RPC port
	DefaultRPCPort = 8545
	// DefaultRPCHost is the default RPC host
	DefaultRPCHost = "localhost"
	// DefaultWSPort is the default WS port
	DefaultWSPort = 8546

	// DefaultPprofListenAddress is the default pprof listen address
	DefaultPprofListenAddress = "localhost:6060"

	// DefaultSystemName is the default system name
	DefaultSystemName = "Gossamer"
	// DefaultSystemVersion is the default system version
	DefaultSystemVersion = "0.3.2"
)

// DefaultRPCModules the default RPC modules
var DefaultRPCModules = []string{
	"system",
	"author",
	"chain",
	"state",
	"rpc",
	"grandpa",
	"offchain",
	"childstate",
	"syncstate",
	"payment",
}

// Config defines the configuration for the gossamer node
type Config struct {
	BaseConfig `mapstructure:",squash"`
	Log        *LogConfig     `mapstructure:"log"`
	Account    *AccountConfig `mapstructure:"account"`
	Core       *CoreConfig    `mapstructure:"core"`
	Network    *NetworkConfig `mapstructure:"network"`
	State      *StateConfig   `mapstructure:"state"`
	RPC        *RPCConfig     `mapstructure:"rpc"`
	Pprof      *PprofConfig   `mapstructure:"pprof"`

	// System holds the system information
	// Do not export this field, as it is not part of the config file
	// and should be set in the source code
	System *SystemConfig
}

// ValidateBasic performs basic validation on the config
func (cfg *Config) ValidateBasic() error {
	if err := cfg.BaseConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("base config: %w", err)
	}
	if err := cfg.Log.ValidateBasic(); err != nil {
		return fmt.Errorf("log config: %w", err)
	}
	if err := cfg.Account.ValidateBasic(); err != nil {
		return fmt.Errorf("account config: %w", err)
	}
	if err := cfg.Core.ValidateBasic(); err != nil {
		return fmt.Errorf("core config: %w", err)
	}
	if err := cfg.Network.ValidateBasic(); err != nil {
		return fmt.Errorf("network config: %w", err)
	}
	if err := cfg.State.ValidateBasic(); err != nil {
		return fmt.Errorf("state config: %w", err)
	}
	if err := cfg.RPC.ValidateBasic(); err != nil {
		return fmt.Errorf("rpc config: %w", err)
	}
	if err := cfg.Pprof.ValidateBasic(); err != nil {
		return fmt.Errorf("pprof config: %w", err)
	}
	return nil
}

// BaseConfig is to marshal/unmarshal toml global config vars
type BaseConfig struct {
	Name               string                      `mapstructure:"name,omitempty"`
	ID                 string                      `mapstructure:"id,omitempty"`
	BasePath           string                      `mapstructure:"base-path,omitempty"`
	ChainSpec          string                      `mapstructure:"chain-spec,omitempty"`
	LogLevel           string                      `mapstructure:"log-level,omitempty"`
	PrometheusPort     uint32                      `mapstructure:"prometheus-port,omitempty"`
	RetainBlocks       uint32                      `mapstructure:"retain-blocks,omitempty"`
	Pruning            pruner.Mode                 `mapstructure:"pruning,omitempty"`
	PrometheusExternal bool                        `mapstructure:"prometheus-external,omitempty"`
	NoTelemetry        bool                        `mapstructure:"no-telemetry"`
	TelemetryURLs      []genesis.TelemetryEndpoint `mapstructure:"telemetry-urls,omitempty"`
}

// SystemConfig represents the system configuration
type SystemConfig struct {
	SystemName    string
	SystemVersion string
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
	Core    string `mapstructure:"core,omitempty"`
	Digest  string `mapstructure:"digest,omitempty"`
	Sync    string `mapstructure:"sync,omitempty"`
	Network string `mapstructure:"network,omitempty"`
	RPC     string `mapstructure:"rpc,omitempty"`
	State   string `mapstructure:"state,omitempty"`
	Runtime string `mapstructure:"runtime,omitempty"`
	Babe    string `mapstructure:"babe,omitempty"`
	Grandpa string `mapstructure:"grandpa,omitempty"`
	Wasmer  string `mapstructure:"wasmer,omitempty"`
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string `mapstructure:"key,omitempty"`
	Unlock string `mapstructure:"unlock,omitempty"`
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port              uint16        `mapstructure:"port"`
	Bootnodes         []string      `mapstructure:"bootnodes"`
	ProtocolID        string        `mapstructure:"protocol"`
	NoBootstrap       bool          `mapstructure:"no-bootstrap"`
	NoMDNS            bool          `mapstructure:"no-mdns"`
	MinPeers          int           `mapstructure:"min-peers"`
	MaxPeers          int           `mapstructure:"max-peers"`
	PersistentPeers   []string      `mapstructure:"persistent-peers"`
	DiscoveryInterval time.Duration `mapstructure:"discovery-interval"`
	PublicIP          string        `mapstructure:"public-ip"`
	PublicDNS         string        `mapstructure:"public-dns"`
	NodeKey           string        `mapstructure:"node-key"`
	ListenAddress     string        `mapstructure:"listen-addr"`
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Role             common.NetworkRole `mapstructure:"role,omitempty"`
	BabeAuthority    bool               `mapstructure:"babe-authority"`
	GrandpaAuthority bool               `mapstructure:"grandpa-authority"`
	WasmInterpreter  string             `mapstructure:"wasm-interpreter,omitempty"`
	GrandpaInterval  time.Duration      `mapstructure:"grandpa-interval,omitempty"`
}

// StateConfig contains the configuration for the state.
type StateConfig struct {
	Rewind uint `mapstructure:"rewind,omitempty"`
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	RPCExternal       bool     `mapstructure:"rpc-external,omitempty"`
	UnsafeRPC         bool     `mapstructure:"unsafe-rpc,omitempty"`
	UnsafeRPCExternal bool     `mapstructure:"unsafe-rpc-external,omitempty"`
	Port              uint32   `mapstructure:"port,omitempty"`
	Host              string   `mapstructure:"host,omitempty"`
	Modules           []string `mapstructure:"modules,omitempty"`
	WSPort            uint32   `mapstructure:"ws-port,omitempty"`
	WSExternal        bool     `mapstructure:"ws-external,omitempty"`
	UnsafeWSExternal  bool     `mapstructure:"unsafe-ws-external,omitempty"`
}

// PprofConfig contains the configuration for Pprof.
type PprofConfig struct {
	Enabled          bool   `mapstructure:"enabled,omitempty"`
	ListeningAddress string `mapstructure:"listening-address,omitempty"`
	BlockProfileRate int    `mapstructure:"block-profile-rate,omitempty"`
	MutexProfileRate int    `mapstructure:"mutex-profile-rate,omitempty"`
}

// ValidateBasic does the basic validation on BaseConfig
func (b *BaseConfig) ValidateBasic() error {
	if b.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if b.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}
	if b.BasePath == "" {
		return fmt.Errorf("base-path directory cannot be empty")
	}
	if b.ChainSpec == "" {
		return fmt.Errorf("chain-spec cannot be empty")
	}
	if b.PrometheusPort == 0 {
		return fmt.Errorf("prometheus port cannot be empty")
	}
	if uint32Max < b.RetainBlocks {
		return fmt.Errorf(
			"retain-blocks value overflows uint32 boundaries, must be less than or equal to: %d",
			uint32Max,
		)
	}

	return nil
}

// ValidateBasic does the basic validation on LogConfig
func (l *LogConfig) ValidateBasic() error {
	return nil
}

// ValidateBasic does the basic validation on AccountConfig
func (a *AccountConfig) ValidateBasic() error {
	if a.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	return nil
}

// ValidateBasic does the basic validation on NetworkConfig
func (n *NetworkConfig) ValidateBasic() error {
	if n.Port == 0 {
		return fmt.Errorf("port cannot be empty")
	}
	if n.ProtocolID == "" {
		return fmt.Errorf("protocol cannot be empty")
	}
	if n.DiscoveryInterval == 0 {
		return fmt.Errorf("discovery-interval cannot be empty")
	}

	return nil
}

// ValidateBasic does the basic validation on CoreConfig
func (c *CoreConfig) ValidateBasic() error {
	if c.WasmInterpreter == "" {
		return fmt.Errorf("wasm-interpreter cannot be empty")
	}
	if c.WasmInterpreter != wazero.Name {
		return fmt.Errorf("wasm-interpreter is invalid")
	}

	return nil
}

// ValidateBasic does the basic validation on StateConfig
func (s *StateConfig) ValidateBasic() error {
	return nil
}

// ValidateBasic does the basic validation on RPCConfig
func (r *RPCConfig) ValidateBasic() error {
	if r.IsRPCEnabled() {
		if r.Port == 0 {
			return fmt.Errorf("port cannot be empty")
		}
		if r.Host == "" {
			return fmt.Errorf("host cannot be empty")
		}
	}
	if r.IsWSEnabled() && r.WSPort == 0 {
		return fmt.Errorf("ws port cannot be empty")
	}

	return nil
}

// ValidateBasic does the basic validation on StateConfig
func (p *PprofConfig) ValidateBasic() error {
	if p.Enabled && p.ListeningAddress == "" {
		return fmt.Errorf("listening address cannot be empty")
	}

	return nil
}

// IsRPCEnabled returns true if RPC is enabled.
func (r *RPCConfig) IsRPCEnabled() bool {
	return r.UnsafeRPCExternal || r.RPCExternal || r.UnsafeRPC
}

// IsWSEnabled returns true if WS is enabled.
func (r *RPCConfig) IsWSEnabled() bool {
	return r.WSExternal || r.UnsafeWSExternal
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: BaseConfig{
			Name:               "Gossamer",
			ID:                 "gssmr",
			BasePath:           defaultBasePath,
			ChainSpec:          "",
			LogLevel:           DefaultLogLevel,
			PrometheusPort:     DefaultPrometheusPort,
			RetainBlocks:       DefaultRetainBlocks,
			Pruning:            DefaultPruning,
			PrometheusExternal: false,
			NoTelemetry:        false,
			TelemetryURLs:      nil,
		},
		Log: &LogConfig{
			Core:    DefaultLogLevel,
			Digest:  DefaultLogLevel,
			Sync:    DefaultLogLevel,
			Network: DefaultLogLevel,
			RPC:     DefaultLogLevel,
			State:   DefaultLogLevel,
			Runtime: DefaultLogLevel,
			Babe:    DefaultLogLevel,
			Grandpa: DefaultLogLevel,
			Wasmer:  DefaultLogLevel,
		},
		Account: &AccountConfig{
			Key:    defaultAccount,
			Unlock: "",
		},
		Core: &CoreConfig{
			Role:             DefaultRole,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			WasmInterpreter:  DefaultWasmInterpreter,
			GrandpaInterval:  DefaultDiscoveryInterval,
		},
		Network: &NetworkConfig{
			Port:              DefaultNetworkPort,
			Bootnodes:         nil,
			ProtocolID:        "/gossamer/gssmr/0",
			NoBootstrap:       false,
			NoMDNS:            true,
			MinPeers:          DefaultMinPeers,
			MaxPeers:          DefaultMaxPeers,
			PersistentPeers:   nil,
			DiscoveryInterval: DefaultDiscoveryInterval,
			PublicIP:          "",
			PublicDNS:         "",
			NodeKey:           "",
			ListenAddress:     "",
		},
		State: &StateConfig{
			Rewind: 0,
		},
		RPC: &RPCConfig{
			RPCExternal:       false,
			UnsafeRPC:         false,
			UnsafeRPCExternal: false,
			Port:              DefaultRPCPort,
			Host:              DefaultRPCHost,
			Modules:           DefaultRPCModules,
			WSPort:            DefaultWSPort,
			WSExternal:        false,
			UnsafeWSExternal:  false,
		},
		Pprof: &PprofConfig{
			Enabled:          false,
			ListeningAddress: DefaultPprofListenAddress,
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &SystemConfig{
			SystemName:    DefaultSystemName,
			SystemVersion: DefaultSystemVersion,
		},
	}
}

// DefaultConfigFromSpec returns the default configuration.
func DefaultConfigFromSpec(nodeSpec *genesis.Genesis) *Config {
	return &Config{
		BaseConfig: BaseConfig{
			Name:               nodeSpec.Name,
			ID:                 nodeSpec.ID,
			BasePath:           defaultBasePath,
			ChainSpec:          "",
			LogLevel:           DefaultLogLevel,
			PrometheusPort:     uint32(9876),
			RetainBlocks:       DefaultRetainBlocks,
			Pruning:            DefaultPruning,
			PrometheusExternal: false,
			NoTelemetry:        false,
			TelemetryURLs:      nil,
		},
		Log: &LogConfig{
			Core:    DefaultLogLevel,
			Digest:  DefaultLogLevel,
			Sync:    DefaultLogLevel,
			Network: DefaultLogLevel,
			RPC:     DefaultLogLevel,
			State:   DefaultLogLevel,
			Runtime: DefaultLogLevel,
			Babe:    DefaultLogLevel,
			Grandpa: DefaultLogLevel,
			Wasmer:  DefaultLogLevel,
		},
		Account: &AccountConfig{
			Key:    defaultAccount,
			Unlock: "",
		},
		Core: &CoreConfig{
			Role:             DefaultRole,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			WasmInterpreter:  DefaultWasmInterpreter,
			GrandpaInterval:  DefaultDiscoveryInterval,
		},
		Network: &NetworkConfig{
			Port:              DefaultNetworkPort,
			Bootnodes:         nodeSpec.Bootnodes,
			ProtocolID:        nodeSpec.ProtocolID,
			NoBootstrap:       false,
			NoMDNS:            false,
			MinPeers:          DefaultMinPeers,
			MaxPeers:          DefaultMaxPeers,
			PersistentPeers:   nil,
			DiscoveryInterval: DefaultDiscoveryInterval,
			PublicIP:          "",
			PublicDNS:         "",
			NodeKey:           "",
			ListenAddress:     "",
		},
		State: &StateConfig{
			Rewind: 0,
		},
		RPC: &RPCConfig{
			RPCExternal:       false,
			UnsafeRPC:         false,
			UnsafeRPCExternal: false,
			Port:              DefaultRPCPort,
			Host:              DefaultRPCHost,
			Modules:           DefaultRPCModules,
			WSPort:            DefaultWSPort,
			WSExternal:        false,
			UnsafeWSExternal:  false,
		},
		Pprof: &PprofConfig{
			Enabled:          false,
			ListeningAddress: DefaultPprofListenAddress,
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &SystemConfig{
			SystemName:    DefaultSystemName,
			SystemVersion: DefaultSystemVersion,
		},
	}
}

// Copy creates a copy of the config.
func Copy(c *Config) Config {
	return Config{
		BaseConfig: BaseConfig{
			Name:               c.BaseConfig.Name,
			ID:                 c.BaseConfig.ID,
			BasePath:           c.BaseConfig.BasePath,
			ChainSpec:          c.BaseConfig.ChainSpec,
			LogLevel:           c.BaseConfig.LogLevel,
			PrometheusPort:     c.PrometheusPort,
			RetainBlocks:       c.RetainBlocks,
			Pruning:            c.Pruning,
			PrometheusExternal: c.PrometheusExternal,
			NoTelemetry:        c.NoTelemetry,
			TelemetryURLs:      c.TelemetryURLs,
		},
		Log: &LogConfig{
			Core:    c.Log.Core,
			Digest:  c.Log.Digest,
			Sync:    c.Log.Sync,
			Network: c.Log.Network,
			RPC:     c.Log.RPC,
			State:   c.Log.State,
			Runtime: c.Log.Runtime,
			Babe:    c.Log.Babe,
			Grandpa: c.Log.Grandpa,
			Wasmer:  c.Log.Wasmer,
		},
		Account: &AccountConfig{
			Key:    c.Account.Key,
			Unlock: c.Account.Unlock,
		},
		Core: &CoreConfig{
			Role:             c.Core.Role,
			BabeAuthority:    c.Core.BabeAuthority,
			GrandpaAuthority: c.Core.GrandpaAuthority,
			WasmInterpreter:  c.Core.WasmInterpreter,
			GrandpaInterval:  c.Core.GrandpaInterval,
		},
		Network: &NetworkConfig{
			Port:              c.Network.Port,
			Bootnodes:         c.Network.Bootnodes,
			ProtocolID:        c.Network.ProtocolID,
			NoBootstrap:       c.Network.NoBootstrap,
			NoMDNS:            c.Network.NoMDNS,
			MinPeers:          c.Network.MinPeers,
			MaxPeers:          c.Network.MaxPeers,
			PersistentPeers:   c.Network.PersistentPeers,
			DiscoveryInterval: c.Network.DiscoveryInterval,
			PublicIP:          c.Network.PublicIP,
			PublicDNS:         c.Network.PublicDNS,
			NodeKey:           c.Network.NodeKey,
			ListenAddress:     c.Network.ListenAddress,
		},
		State: &StateConfig{
			Rewind: c.State.Rewind,
		},
		RPC: &RPCConfig{
			UnsafeRPC:         c.RPC.UnsafeRPC,
			UnsafeRPCExternal: c.RPC.UnsafeRPCExternal,
			RPCExternal:       c.RPC.RPCExternal,
			Port:              c.RPC.Port,
			Host:              c.RPC.Host,
			Modules:           c.RPC.Modules,
			WSPort:            c.RPC.WSPort,
			WSExternal:        c.RPC.WSExternal,
			UnsafeWSExternal:  c.RPC.UnsafeWSExternal,
		},
		Pprof: &PprofConfig{
			Enabled:          c.Pprof.Enabled,
			ListeningAddress: c.Pprof.ListeningAddress,
			BlockProfileRate: c.Pprof.BlockProfileRate,
			MutexProfileRate: c.Pprof.MutexProfileRate,
		},
		System: &SystemConfig{
			SystemName:    c.System.SystemName,
			SystemVersion: c.System.SystemVersion,
		},
	}
}

// EnsureRoot creates the root, config, and data directories if they don't exist,
// and returns error if it fails.
func EnsureRoot(basePath string) error {
	if err := os.EnsureDir(basePath, DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create root directory: %w", err)
	}
	if err := os.EnsureDir(filepath.Join(basePath, defaultConfigDir), DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// Chain is a string representing a chain
type Chain string

const (
	// PolkadotChain is the Polkadot chain
	PolkadotChain Chain = "polkadot"
	// KusamaChain is the Kusama chain
	KusamaChain Chain = "kusama"
	// WestendChain is the Westend chain
	WestendChain Chain = "westend"
	// WestendDevChain is the Westend dev chain
	WestendDevChain Chain = "westend-dev"
	// WestendLocalChain is the Westend local chain
	WestendLocalChain Chain = "westend-local"
)

// String returns the string representation of the chain
func (c Chain) String() string {
	return string(c)
}

// NetworkRole is a string representing a network role
type NetworkRole string

const (
	// NoNetworkRole is no network role
	NoNetworkRole NetworkRole = "none"

	// FullNode is a full node
	FullNode NetworkRole = "full"

	// LightNode is a light node
	LightNode NetworkRole = "light"

	// AuthorityNode is an authority node
	AuthorityNode NetworkRole = "authority"
)

// String returns the string representation of the network role
func (n NetworkRole) String() string {
	return string(n)
}

// GetChainSpec returns the path to the chain-spec file.
func GetChainSpec(basePath string) string {
	return filepath.Join(basePath, defaultChainSpecFile)
}
