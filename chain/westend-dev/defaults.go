// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westenddev

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"time"
)

// DefaultConfig returns a westend dev node configuration
func DefaultConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Westend",
			ID:             "westend_dev",
			BasePath:       "~/.gossamer/westend-dev",
			Genesis:        "./chain/westend-dev/westend-dev-spec-raw.json",
			LogLevel:       "info",
			MetricsAddress: ":9876",
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
		},
		State: &cfg.StateConfig{
			Rewind: 0,
		},
		Network: &cfg.NetworkConfig{
			Port:              7001,
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
		},
		RPC: &cfg.RPCConfig{
			WS:      false,
			Enabled: false,
			Port:    8545,
			Host:    "localhost",
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
