// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"github.com/ChainSafe/gossamer/dot/config/toml"
)

// Default returns a default TOML configuration for Gossamer.
func Default() toml.Config {
	return toml.Config{
		Global: toml.GlobalConfig{
			Name:           "Gossamer",
			ID:             "gssmr",
			LogLvl:         "info",
			MetricsAddress: "localhost:9876",
			RetainBlocks:   256,
			Pruning:        "archive",
		},
		Log: toml.LogConfig{
			CoreLvl: "info",
			SyncLvl: "info",
		},
		Account: toml.AccountConfig{
			Key:    "",
			Unlock: "",
		},
		Core: toml.CoreConfig{
			Roles:            4,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  1,
		},
		Network: toml.NetworkConfig{
			Bootnodes:   nil,
			ProtocolID:  "/gossamer/gssmr/0",
			NoBootstrap: false,
			NoMDNS:      false,
			MinPeers:    1,
			MaxPeers:    3,
		},
		RPC: toml.RPCConfig{
			Enabled:  true,
			Unsafe:   true,
			WSUnsafe: true,
			Host:     "localhost",
			Modules:  []string{"system", "author", "chain", "state", "dev", "rpc"},
			WS:       false,
		},
	}
}
