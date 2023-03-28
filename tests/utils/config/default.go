// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

// Default returns a default TOML configuration for Gossamer.
func Default() cfg.Config {
	return cfg.Config{
		BaseConfig: &cfg.BaseConfig{
			Name:           "Gossamer",
			ID:             "gssmr",
			LogLevel:       "info",
			MetricsAddress: "localhost:9876",
			RetainBlocks:   256,
			Pruning:        "archive",
		},
		Log: &cfg.LogConfig{
			Core: "info",
			Sync: "info",
		},
		Account: &cfg.AccountConfig{
			Key:    "",
			Unlock: "",
		},
		Core: &cfg.CoreConfig{
			Role:             4,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  1,
		},
		Network: &cfg.NetworkConfig{
			Bootnodes:   nil,
			ProtocolID:  "/gossamer/gssmr/0",
			NoBootstrap: false,
			NoMDNS:      false,
			MinPeers:    1,
			MaxPeers:    3,
		},
		RPC: &cfg.RPCConfig{
			Enabled:  true,
			Unsafe:   true,
			WSUnsafe: true,
			Host:     "localhost",
			Modules:  []string{"system", "author", "chain", "state", "dev", "rpc"},
			WS:       false,
		},
		State: &cfg.StateConfig{},
	}
}
