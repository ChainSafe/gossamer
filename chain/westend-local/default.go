// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westendlocal

import (
	"time"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

const (
	// DefaultBasePathAlice is the default basepath for the westend local alice node
	DefaultBasePathAlice = "~/.gossamer/westend-local-alice"
	// DefaultBasePathBob is the default basepath for the westend local bob node
	DefaultBasePathBob = "~/.gossamer/westend-local-bob"
	// DefaultBasePathCharlie is the default basepath for the westend local charlie node
	DefaultBasePathCharlie = "~/.gossamer/westend-local-charlie"
)

// DefaultAliceConfig returns a westend local node configuration
func DefaultAliceConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Westend",
			ID:             "westend_local",
			BasePath:       DefaultBasePathAlice,
			ChainSpec:      "./chain/westend-local/westend-local-spec-raw.json",
			LogLevel:       "info",
			PrometheusPort: ":9876",
			RetainBlocks:   512,
			Pruning:        pruner.Archive,
		},
		Log: &cfg.LogConfig{
			Core:    "info",
			Digest:  "info",
			Sync:    "info",
			Network: "info",
			RPC:     "info",
			State:   "info",
			Runtime: "info",
			Babe:    "info",
			Grandpa: "info",
			Wasmer:  "info",
		},
		Account: &cfg.AccountConfig{
			Key:    "alice",
			Unlock: "",
		},
		Core: &cfg.CoreConfig{
			Role:             common.AuthorityRole,
			WasmInterpreter:  wasmer.Name,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  time.Second,
			BABELead:         true,
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:              7001,
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
			ProtocolID:        "dot",
		},
		RPC: &cfg.RPCConfig{
			Port: 8545,
			Host: "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
			WSPort: 8546,
		},
		Pprof: &cfg.PprofConfig{
			ListeningAddress: "localhost:6060",
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &cfg.SystemConfig{
			SystemName:    "gossamer",
			SystemVersion: "0.1.0",
		},
	}
}

// DefaultBobConfig returns a westend local node configuration with bob as the authority
func DefaultBobConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Westend",
			ID:             "westend_local",
			BasePath:       DefaultBasePathBob,
			ChainSpec:      "./chain/westend-local/westend-local-spec-raw.json",
			LogLevel:       "info",
			PrometheusPort: ":9986",
			RetainBlocks:   512,
			Pruning:        pruner.Archive,
		},
		Log: &cfg.LogConfig{
			Core:    "info",
			Digest:  "info",
			Sync:    "info",
			Network: "info",
			RPC:     "info",
			State:   "info",
			Runtime: "info",
			Babe:    "info",
			Grandpa: "info",
			Wasmer:  "info",
		},
		Account: &cfg.AccountConfig{
			Key:    "bob",
			Unlock: "",
		},
		Core: &cfg.CoreConfig{
			Role:             common.AuthorityRole,
			WasmInterpreter:  wasmer.Name,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  time.Second,
			BABELead:         true,
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:              7011,
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
		},
		RPC: &cfg.RPCConfig{
			Port: 8555,
			Host: "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
			WSPort: 8546,
		},
		Pprof: &cfg.PprofConfig{
			ListeningAddress: "localhost:6070",
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &cfg.SystemConfig{
			SystemName:    "gossamer",
			SystemVersion: "0.1.0",
		},
	}
}

// DefaultCharlieConfig returns a westend local node configuration with charlie as the authority
func DefaultCharlieConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Westend",
			ID:             "westend_local",
			BasePath:       DefaultBasePathCharlie,
			ChainSpec:      "./chain/westend-local/westend-local-spec-raw.json",
			LogLevel:       "info",
			PrometheusPort: ":9996",
			RetainBlocks:   512,
			Pruning:        pruner.Archive,
		},
		Log: &cfg.LogConfig{
			Core:    "info",
			Digest:  "info",
			Sync:    "info",
			Network: "info",
			RPC:     "info",
			State:   "info",
			Runtime: "info",
			Babe:    "info",
			Grandpa: "info",
			Wasmer:  "info",
		},
		Account: &cfg.AccountConfig{
			Key:    "charlie",
			Unlock: "",
		},
		Core: &cfg.CoreConfig{
			Role:             common.AuthorityRole,
			WasmInterpreter:  wasmer.Name,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  time.Second,
			BABELead:         true,
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:              7021,
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
		},
		RPC: &cfg.RPCConfig{
			Port: 8565,
			Host: "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
			WSPort: 8546,
		},
		Pprof: &cfg.PprofConfig{
			ListeningAddress: "localhost:6080",
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &cfg.SystemConfig{
			SystemName:    "gossamer",
			SystemVersion: "0.1.0",
		},
	}
}

// DefaultConfig returns a westend local node configuration
func DefaultConfig() *cfg.Config {
	return DefaultAliceConfig()
}
