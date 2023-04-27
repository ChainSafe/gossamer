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
	config := cfg.Copy(DefaultConfig())
	config.BasePath = DefaultBasePathBob
	config.PrometheusPort = uint32(9856)
	config.Network.Port = 7001
	config.RPC.Port = 8545
	config.RPC.WSPort = 8546
	config.Pprof.ListeningAddress = "localhost:6060"

	return &config
}

// DefaultBobConfig returns a westend local node configuration with bob as the authority
func DefaultBobConfig() *cfg.Config {
	config := cfg.Copy(DefaultConfig())
	config.BasePath = DefaultBasePathBob
	config.PrometheusPort = uint32(9866)
	config.Network.Port = 7011
	config.RPC.Port = 8555
	config.RPC.WSPort = 8556
	config.Pprof.ListeningAddress = "localhost:6070"

	return &config
}

// DefaultCharlieConfig returns a westend local node configuration with charlie as the authority
func DefaultCharlieConfig() *cfg.Config {
	config := cfg.Copy(DefaultConfig())
	config.BasePath = DefaultBasePathCharlie
	config.PrometheusPort = uint32(9876)
	config.Network.Port = 7021
	config.RPC.Port = 8565
	config.RPC.WSPort = 8566
	config.Pprof.ListeningAddress = "localhost:6080"

	return &config
}

// DefaultConfig returns a westend local node configuration
func DefaultConfig() *cfg.Config {
	return &cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Westend",
			ID:             "westend_local",
			ChainSpec:      "./chain/westend-local/westend-local-spec-raw.json",
			LogLevel:       "info",
			PrometheusPort: uint32(9876),
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
			Key:    "",
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
			NoBootstrap:       false,
			NoMDNS:            false,
			DiscoveryInterval: 10 * time.Second,
			MinPeers:          cfg.DefaultMinPeers,
			MaxPeers:          cfg.DefaultMaxPeers,
		},
		RPC: &cfg.RPCConfig{
			Host: "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
			WSPort: 8546,
		},
		Pprof: &cfg.PprofConfig{
			BlockProfileRate: 0,
			MutexProfileRate: 0,
		},
		System: &cfg.SystemConfig{
			SystemName:    cfg.DefaultSystemName,
			SystemVersion: cfg.DefaultSystemVersion,
		},
	}
}
