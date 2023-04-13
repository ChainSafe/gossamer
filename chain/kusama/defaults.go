// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

var (
	// DefaultName Default node name
	DefaultName = "Kusama"
	// DefaultID Default chain ID
	DefaultID = "ksmcc3"
	// DefaultBasePath Default node base directory path
	DefaultBasePath = "~/.gossamer/kusama"

	// DefaultPrometheusPort is the default metrics server listening address.
	DefaultPrometheusPort = uint32(9876)

	// DefaultLvl is the default log level
	DefaultLvl = "info"

	// DefaultPruningMode is the default pruning mode
	DefaultPruningMode = pruner.Mode("archive")
	// DefaultRetainBlocks is the default retained blocks
	DefaultRetainBlocks = uint32(512)

	// DefaultTelemetryURLs is the default URL of the telemetry server to connect to.
	DefaultTelemetryURLs = []genesis.TelemetryEndpoint(nil)

	// InitConfig

	// DefaultChainSpec is the default chain-spec json path
	DefaultChainSpec = "./chain/kusama/genesis.json"

	// AccountConfig

	// DefaultKey Default account key
	DefaultKey = ""
	// DefaultUnlock Default account unlock
	DefaultUnlock = ""

	// CoreConfig

	// DefaultAuthority true if BABE block producer
	DefaultAuthority = false
	// DefaultRole Default node roles
	DefaultRole = common.FullNodeRole // full node (see Table D.2)
	// DefaultWasmInterpreter is the name of the wasm interpreter to use by default
	DefaultWasmInterpreter = wasmer.Name

	// NetworkConfig

	// DefaultNetworkPort network port
	DefaultNetworkPort = uint16(7001)
	// DefaultNetworkBootnodes network bootnodes
	DefaultNetworkBootnodes []string
	// DefaultNoBootstrap disables bootstrap
	DefaultNoBootstrap = false
	// DefaultNoMDNS disables mDNS discovery
	DefaultNoMDNS = false

	// RPCConfig

	// DefaultRPCHTTPHost rpc host
	DefaultRPCHTTPHost = "localhost"
	// DefaultRPCHTTPPort rpc port
	DefaultRPCHTTPPort = uint32(8545)
	// DefaultRPCModules rpc modules
	DefaultRPCModules = []string{
		"system", "author", "chain",
		"state", "rpc", "grandpa",
		"offchain", "childstate", "syncstate",
		"payment",
	}
	// DefaultRPCWSPort rpc websocket port
	DefaultRPCWSPort = uint32(8546)
)

const (
	// PprofConfig

	// DefaultPprofListeningAddress default pprof HTTP server listening address.
	DefaultPprofListeningAddress = "localhost:6060"

	// DefaultPprofBlockRate default block profile rate.
	// Set to 0 to disable profiling.
	DefaultPprofBlockRate = 0

	// DefaultPprofMutexRate default mutex profile rate.
	// Set to 0 to disable profiling.
	DefaultPprofMutexRate = 0
)

// DefaultConfig returns a kusama node configuration
func DefaultConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           DefaultName,
			ID:             DefaultID,
			BasePath:       DefaultBasePath,
			ChainSpec:      DefaultChainSpec,
			LogLevel:       DefaultLvl,
			PrometheusPort: DefaultPrometheusPort,
			RetainBlocks:   DefaultRetainBlocks,
			Pruning:        DefaultPruningMode,
			TelemetryURLs:  DefaultTelemetryURLs,
		},
		Log: &cfg.LogConfig{
			Core:    DefaultLvl,
			Digest:  DefaultLvl,
			Sync:    DefaultLvl,
			Network: DefaultLvl,
			RPC:     DefaultLvl,
			State:   DefaultLvl,
			Runtime: DefaultLvl,
			Babe:    DefaultLvl,
			Grandpa: DefaultLvl,
			Wasmer:  DefaultLvl,
		},
		Account: &cfg.AccountConfig{
			Key:    DefaultKey,
			Unlock: DefaultUnlock,
		},
		Core: &cfg.CoreConfig{
			Role:            DefaultRole,
			WasmInterpreter: DefaultWasmInterpreter,
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:        DefaultNetworkPort,
			Bootnodes:   DefaultNetworkBootnodes,
			NoBootstrap: DefaultNoBootstrap,
			NoMDNS:      DefaultNoMDNS,
		},
		RPC: &cfg.RPCConfig{
			Port:    DefaultRPCHTTPPort,
			Host:    DefaultRPCHTTPHost,
			Modules: DefaultRPCModules,
			WSPort:  DefaultRPCWSPort,
		},
		Pprof: &cfg.PprofConfig{
			ListeningAddress: DefaultPprofListeningAddress,
			BlockProfileRate: DefaultPprofBlockRate,
			MutexProfileRate: DefaultPprofMutexRate,
		},
		System: &cfg.SystemConfig{
			SystemName:    "gossamer",
			SystemVersion: "0.1.0",
		},
	}
}
