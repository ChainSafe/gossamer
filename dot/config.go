// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package dot

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/chain/gssmr"
	"github.com/ChainSafe/gossamer/chain/ksmcc"
	"github.com/ChainSafe/gossamer/chain/polkadot"
	"github.com/ChainSafe/gossamer/dot/types"
	log "github.com/ChainSafe/log15"
)

// TODO: create separate types for toml config and internal config, needed since we don't want to expose all
// the internal config options, also type conversions might be needed from toml -> internal types

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
}

// GlobalConfig is used for every node command
type GlobalConfig struct {
	Name     string
	ID       string
	BasePath string
	LogLvl   log.Lvl
}

// LogConfig represents the log levels for individual packages
type LogConfig struct {
	CoreLvl           log.Lvl
	SyncLvl           log.Lvl
	NetworkLvl        log.Lvl
	RPCLvl            log.Lvl
	StateLvl          log.Lvl
	RuntimeLvl        log.Lvl
	BlockProducerLvl  log.Lvl
	FinalityGadgetLvl log.Lvl
}

// InitConfig is the configuration for the node initialization
type InitConfig struct {
	GenesisRaw string
}

// AccountConfig is to marshal/unmarshal account config vars
type AccountConfig struct {
	Key    string // TODO: change to array
	Unlock string // TODO: change to array
}

// NetworkConfig is to marshal/unmarshal toml network config vars
type NetworkConfig struct {
	Port        uint32
	Bootnodes   []string
	ProtocolID  string
	NoBootstrap bool
	NoMDNS      bool
	MinPeers    int
	MaxPeers    int
}

// CoreConfig is to marshal/unmarshal toml core config vars
type CoreConfig struct {
	Roles                    byte
	BabeAuthority            bool
	GrandpaAuthority         bool
	BabeThresholdNumerator   uint64
	BabeThresholdDenominator uint64
	SlotDuration             uint64
	EpochLength              uint64
	WasmInterpreter          string
}

// RPCConfig is to marshal/unmarshal toml RPC config vars
type RPCConfig struct {
	Enabled    bool
	External   bool
	Port       uint32
	Host       string
	Modules    []string
	WSPort     uint32
	WS         bool
	WSExternal bool
}

// String will return the json representation for a Config
func (c *Config) String() string {
	out, _ := json.MarshalIndent(c, "", "\t")
	return string(out)
}

// networkServiceEnabled returns true if the network service is enabled
func networkServiceEnabled(cfg *Config) bool {
	return cfg.Core.Roles != byte(0)
}

// GssmrConfig returns a new test configuration using the provided basepath
func GssmrConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:     gssmr.DefaultName,
			ID:       gssmr.DefaultID,
			BasePath: gssmr.DefaultBasePath,
			LogLvl:   gssmr.DefaultLvl,
		},
		Log: LogConfig{
			CoreLvl:           gssmr.DefaultLvl,
			SyncLvl:           gssmr.DefaultLvl,
			NetworkLvl:        gssmr.DefaultLvl,
			RPCLvl:            gssmr.DefaultLvl,
			StateLvl:          gssmr.DefaultLvl,
			RuntimeLvl:        gssmr.DefaultLvl,
			BlockProducerLvl:  gssmr.DefaultLvl,
			FinalityGadgetLvl: gssmr.DefaultLvl,
		},
		Init: InitConfig{
			GenesisRaw: gssmr.DefaultGenesisRaw,
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
		},
		Network: NetworkConfig{
			Port:        gssmr.DefaultNetworkPort,
			Bootnodes:   gssmr.DefaultNetworkBootnodes,
			NoBootstrap: gssmr.DefaultNoBootstrap,
			NoMDNS:      gssmr.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    gssmr.DefaultRPCHTTPPort,
			Host:    gssmr.DefaultRPCHTTPHost,
			Modules: gssmr.DefaultRPCModules,
			WSPort:  gssmr.DefaultRPCWSPort,
		},
	}
}

// KsmccConfig returns a "ksmcc" node configuration
func KsmccConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:     ksmcc.DefaultName,
			ID:       ksmcc.DefaultID,
			BasePath: ksmcc.DefaultBasePath,
			LogLvl:   ksmcc.DefaultLvl,
		},
		Log: LogConfig{
			CoreLvl:           ksmcc.DefaultLvl,
			SyncLvl:           ksmcc.DefaultLvl,
			NetworkLvl:        ksmcc.DefaultLvl,
			RPCLvl:            ksmcc.DefaultLvl,
			StateLvl:          ksmcc.DefaultLvl,
			RuntimeLvl:        ksmcc.DefaultLvl,
			BlockProducerLvl:  ksmcc.DefaultLvl,
			FinalityGadgetLvl: ksmcc.DefaultLvl,
		},
		Init: InitConfig{
			GenesisRaw: ksmcc.DefaultGenesisRaw,
		},
		Account: AccountConfig{
			Key:    ksmcc.DefaultKey,
			Unlock: ksmcc.DefaultUnlock,
		},
		Core: CoreConfig{
			Roles:           ksmcc.DefaultRoles,
			WasmInterpreter: ksmcc.DefaultWasmInterpreter,
		},
		Network: NetworkConfig{
			Port:        ksmcc.DefaultNetworkPort,
			Bootnodes:   ksmcc.DefaultNetworkBootnodes,
			NoBootstrap: ksmcc.DefaultNoBootstrap,
			NoMDNS:      ksmcc.DefaultNoMDNS,
		},
		RPC: RPCConfig{
			Port:    ksmcc.DefaultRPCHTTPPort,
			Host:    ksmcc.DefaultRPCHTTPHost,
			Modules: ksmcc.DefaultRPCModules,
			WSPort:  ksmcc.DefaultRPCWSPort,
		},
	}
}

// PolkadotConfig returns a "polkadot" node configuration
func PolkadotConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			Name:     polkadot.DefaultName,
			ID:       polkadot.DefaultID,
			BasePath: polkadot.DefaultBasePath,
			LogLvl:   polkadot.DefaultLvl,
		},
		Log: LogConfig{
			CoreLvl:           polkadot.DefaultLvl,
			SyncLvl:           polkadot.DefaultLvl,
			NetworkLvl:        polkadot.DefaultLvl,
			RPCLvl:            polkadot.DefaultLvl,
			StateLvl:          polkadot.DefaultLvl,
			RuntimeLvl:        polkadot.DefaultLvl,
			BlockProducerLvl:  polkadot.DefaultLvl,
			FinalityGadgetLvl: polkadot.DefaultLvl,
		},
		Init: InitConfig{
			GenesisRaw: polkadot.DefaultGenesisRaw,
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
	}
}
