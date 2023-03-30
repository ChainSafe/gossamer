// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/os"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"path/filepath"
	"time"
)

const uint32Max = ^uint32(0)

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
	Name           string                      `mapstructure:"name,omitempty"`
	ID             string                      `mapstructure:"id,omitempty"`
	BasePath       string                      `mapstructure:"base-path,omitempty"`
	Genesis        string                      `mapstructure:"genesis,omitempty"`
	LogLevel       string                      `mapstructure:"log-level,omitempty"`
	MetricsAddress string                      `mapstructure:"metrics-address,omitempty"`
	RetainBlocks   uint32                      `mapstructure:"retain-blocks,omitempty"`
	Pruning        pruner.Mode                 `mapstructure:"pruning,omitempty"`
	PublishMetrics bool                        `mapstructure:"publish-metrics"`
	NoTelemetry    bool                        `mapstructure:"no-telemetry"`
	TelemetryURLs  []genesis.TelemetryEndpoint `mapstructure:"telemetry-urls,omitempty"`
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
	ListenAddress     string        `mapstructure:"listen-address"`
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Role             common.NetworkRole `mapstructure:"role,omitempty"`
	BabeAuthority    bool               `mapstructure:"babe-authority"`
	GrandpaAuthority bool               `mapstructure:"grandpa-authority"`
	SlotDuration     uint64             `mapstructure:"slot-duration,omitempty"`
	EpochLength      uint64             `mapstructure:"epoch-length,omitempty"`
	WasmInterpreter  string             `mapstructure:"wasm-interpreter,omitempty"`
	GrandpaInterval  time.Duration      `mapstructure:"grandpa-interval,omitempty"`
	BABELead         bool               `mapstructure:"babe-lead,omitempty"`
}

// StateConfig contains the configuration for the state.
type StateConfig struct {
	Rewind uint `mapstructure:"rewind,omitempty"`
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled          bool     `mapstructure:"enabled,omitempty"`
	Unsafe           bool     `mapstructure:"unsafe,omitempty"`
	UnsafeExternal   bool     `mapstructure:"unsafe-external,omitempty"`
	External         bool     `mapstructure:"external,omitempty"`
	Port             uint32   `mapstructure:"port,omitempty"`
	Host             string   `mapstructure:"host,omitempty"`
	Modules          []string `mapstructure:"modules,omitempty"`
	WSPort           uint32   `mapstructure:"ws-port,omitempty"`
	WS               bool     `mapstructure:"ws,omitempty"`
	WSExternal       bool     `mapstructure:"ws-external,omitempty"`
	WSUnsafe         bool     `mapstructure:"ws-unsafe,omitempty"`
	WSUnsafeExternal bool     `mapstructure:"ws-unsafe-external,omitempty"`
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
	if b.Genesis == "" {
		return fmt.Errorf("genesis cannot be empty")
	}
	if b.MetricsAddress == "" {
		return fmt.Errorf("metrics address cannot be empty")
	}
	if uint32Max < b.RetainBlocks {
		return fmt.Errorf("retain-blocks value overflows uint32 boundaries, must be less than or equal to: %d", uint32Max)
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
	//if a.Unlock == "" {
	//	return fmt.Errorf("unlock cannot be empty")
	//}

	return nil
}

// ValidateBasic does the basic validation on NetworkConfig
func (n *NetworkConfig) ValidateBasic() error {
	if n.Port == 0 {
		return fmt.Errorf("port cannot be empty")
	}
	//if n.ProtocolID == "" {
	//	return fmt.Errorf("protocol cannot be empty")
	//}
	//if n.MinPeers == 0 {
	//	return fmt.Errorf("minimum-peers cannot be empty")
	//}
	//if n.MaxPeers == 0 {
	//	return fmt.Errorf("maximum-peers cannot be empty")
	//}
	if n.DiscoveryInterval == 0 {
		return fmt.Errorf("discovery-interval cannot be empty")
	}
	//if n.PublicIP == "" {
	//	return fmt.Errorf("public IP cannot be empty")
	//}
	//if n.PublicDNS == "" {
	//	return fmt.Errorf("public DNS cannot be empty")
	//}

	return nil
}

// ValidateBasic does the basic validation on CoreConfig
func (c *CoreConfig) ValidateBasic() error {
	//if c.SlotDuration == 0 {
	//	return fmt.Errorf("slot duration cannot be empty")
	//}
	//if c.EpochLength == 0 {
	//	return fmt.Errorf("epoch length cannot be empty")
	//}
	if c.WasmInterpreter == "" {
		return fmt.Errorf("wasm interpreter cannot be empty")
	}
	if c.GrandpaInterval == 0 {
		return fmt.Errorf("grandpa interval cannot be empty")
	}
	if c.WasmInterpreter == "" {
		return fmt.Errorf("wasm-interpreter cannot be empty")
	}
	if c.WasmInterpreter != wasmer.Name {
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
	if r.Enabled {
		if r.Port == 0 {
			return fmt.Errorf("port cannot be empty")
		}
		if r.Host == "" {
			return fmt.Errorf("host cannot be empty")
		}
	}
	if r.WS && r.WSPort == 0 {
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
	return r.Enabled || r.External || r.Unsafe || r.UnsafeExternal
}

// IsWSEnabled returns true if WS is enabled.
func (r *RPCConfig) IsWSEnabled() bool {
	return r.WS || r.WSExternal || r.WSUnsafe || r.WSUnsafeExternal
}

// EnsureRoot creates the root, config, and data directories if they don't exist,
// and returns error if it fails.
func EnsureRoot(rootDir string, config *Config) error {
	if err := os.EnsureDir(rootDir, DefaultDirPerm); err != nil {
		panic(err.Error())
	}
	if err := os.EnsureDir(filepath.Join(rootDir, defaultConfigDir), DefaultDirPerm); err != nil {
		panic(err.Error())
	}

	configFilePath := filepath.Join(rootDir, defaultConfigFilePath)
	fmt.Println("babe-lead", config.Core.BABELead)
	fmt.Println("configFilePath", configFilePath)
	return WriteConfigFile(configFilePath, config)
}
