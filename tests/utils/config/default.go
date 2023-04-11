// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"time"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
)

// Default returns a default TOML configuration for Gossamer.
func Default() cfg.Config {
	return cfg.Config{
		BaseConfig: cfg.BaseConfig{
			Name:           "Gossamer",
			ID:             "gssmr",
			LogLevel:       "info",
			MetricsAddress: "localhost:9876",
			RetainBlocks:   256,
			Pruning:        "archive",
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
			Role:             4,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  1 * time.Second,
			WasmInterpreter:  wasmer.Name,
		},
		Network: &cfg.NetworkConfig{
			Bootnodes:         nil,
			ProtocolID:        "/gossamer/gssmr/0",
			NoBootstrap:       false,
			NoMDNS:            false,
			MinPeers:          1,
			MaxPeers:          3,
			DiscoveryInterval: time.Second * 1,
		},
		RPC: &cfg.RPCConfig{
			UnsafeRPC:         true,
			UnsafeRPCExternal: true,
			UnsafeWSExternal:  true,
			Host:              "localhost",
			Modules: []string{
				"system", "author", "chain", "state", "rpc",
				"grandpa", "offchain", "childstate", "syncstate", "payment"},
		},
		State:  &cfg.StateConfig{},
		Pprof:  &cfg.PprofConfig{},
		System: &cfg.SystemConfig{},
	}
}
