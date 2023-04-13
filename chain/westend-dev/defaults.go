// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westenddev

import (
	"time"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

const (
	// DefaultName is the default node name
	DefaultName = "Westend"
	// DefaultID is the default node ID
	DefaultID = "westend_dev"
	// DefaultBasePath is the default basepath for the westend dev node
	DefaultBasePath = "~/.gossamer/westend-dev"
	// DefaultChainSpec is the default chain spec for the westend dev node
	DefaultChainSpec = "./chain/westend-dev/westend-dev-spec-raw.json"
	// DefaultLogLevel is the default log level for the westend dev node
	DefaultLogLevel = "info"
	// DefaultPrometheusPort is the default metrics port for the westend dev node
	DefaultPrometheusPort = ":9876"
	// DefaultRetainBlocks is the default number of blocks to retain
	DefaultRetainBlocks = uint32(512)
	// DefaultPruning is the default pruning mode
	DefaultPruning = pruner.Archive

	// DefaultRole is the default node role
	DefaultRole = common.AuthorityRole
	// DefaultWasmInterpreter is the default wasm interpreter
	DefaultWasmInterpreter = wasmer.Name
	// DefaultBabeAuthority is the default babe authority
	DefaultBabeAuthority = true
	// DefaultGrandpaAuthority is the default grandpa authority
	DefaultGrandpaAuthority = true
	// DefaultGrandpaInterval is the default grandpa interval
	DefaultGrandpaInterval = time.Second
	// DefaultProtocolID is the default protocol ID
	DefaultProtocolID = "dot"

	// DefaultPort is the default port for the westend dev node
	DefaultPort = 7001
	// DefaultNoBootstrap is the default bootstrap flag for the westend dev node
	DefaultNoBootstrap = true
	// DefaultNoMDNS is the default mdns flag for the westend dev node
	DefaultNoMDNS = true
	// DefaultDiscoveryInterval is the default discovery interval for the westend dev node
	DefaultDiscoveryInterval = time.Second

	// DefaultRPCPort is the default rpc port for the westend dev node
	DefaultRPCPort = uint32(8545)
	// DefaultRPCHost is the default rpc host for the westend dev node
	DefaultRPCHost = "localhost"
	// DefaultWSPort is the default websocket port for the westend dev node
	DefaultWSPort = uint32(8546)

	// DefaultPPROFEnabled is the default pprof flag for the westend dev node
	DefaultPPROFEnabled = false
	// DefaultPprofListeningAddress is the default pprof listening address for the westend dev node
	DefaultPprofListeningAddress = "localhost:6060"
	// DefaultPPROFBlockProfileRate is the default pprof profile rate for the westend dev node
	DefaultPPROFBlockProfileRate = 0
	// DefaultPPROFMutexProfileRate is the default pprof mutex profile rate for the westend dev node
	DefaultPPROFMutexProfileRate = 0
)

var (
	// DefaultRPCModules is the default rpc modules for the westend dev node
	DefaultRPCModules = []string{"system",
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
)

// DefaultConfig returns a westend dev node configuration
func DefaultConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           DefaultName,
			ID:             DefaultID,
			BasePath:       DefaultBasePath,
			ChainSpec:      DefaultChainSpec,
			LogLevel:       DefaultLogLevel,
			PrometheusPort: uint32(9876),
			RetainBlocks:   DefaultRetainBlocks,
			Pruning:        DefaultPruning,
		},
		Log: &cfg.LogConfig{
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
		Account: &cfg.AccountConfig{
			Key:    "",
			Unlock: "",
		},
		Core: &cfg.CoreConfig{
			Role:             DefaultRole,
			WasmInterpreter:  DefaultWasmInterpreter,
			BabeAuthority:    DefaultBabeAuthority,
			GrandpaAuthority: DefaultGrandpaAuthority,
			GrandpaInterval:  DefaultGrandpaInterval,
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:              DefaultPort,
			NoBootstrap:       DefaultNoBootstrap,
			NoMDNS:            DefaultNoMDNS,
			DiscoveryInterval: DefaultDiscoveryInterval,
			ProtocolID:        DefaultProtocolID,
			MinPeers:          cfg.DefaultMinPeers,
			MaxPeers:          cfg.DefaultMaxPeers,
		},
		RPC: &cfg.RPCConfig{
			Port:    DefaultRPCPort,
			Host:    DefaultRPCHost,
			Modules: DefaultRPCModules,
			WSPort:  DefaultWSPort,
		},
		Pprof: &cfg.PprofConfig{
			Enabled:          DefaultPPROFEnabled,
			ListeningAddress: DefaultPprofListeningAddress,
			BlockProfileRate: DefaultPPROFBlockProfileRate,
			MutexProfileRate: DefaultPPROFMutexProfileRate,
		},
		System: &cfg.SystemConfig{
			SystemName:    "gossamer",
			SystemVersion: "0.1.0",
		},
	}
}
