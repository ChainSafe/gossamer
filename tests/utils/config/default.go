// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
)

func generateDefaultConfig() *ctoml.Config {
	return &ctoml.Config{
		Global: ctoml.GlobalConfig{
			Name:           "Gossamer",
			ID:             "gssmr",
			LogLvl:         "crit",
			MetricsAddress: "localhost:9876",
			RetainBlocks:   256,
			Pruning:        "archive",
		},
		Log: ctoml.LogConfig{
			CoreLvl: "info",
			SyncLvl: "info",
		},
		Init: ctoml.InitConfig{
			Genesis: "./chain/gssmr/genesis.json",
		},
		Account: ctoml.AccountConfig{
			Key:    "",
			Unlock: "",
		},
		Core: ctoml.CoreConfig{
			Roles:            4,
			BabeAuthority:    true,
			GrandpaAuthority: true,
			GrandpaInterval:  1,
		},
		Network: ctoml.NetworkConfig{
			Bootnodes:   nil,
			ProtocolID:  "/gossamer/gssmr/0",
			NoBootstrap: false,
			NoMDNS:      false,
			MinPeers:    1,
			MaxPeers:    3,
		},
		RPC: ctoml.RPCConfig{
			Enabled:  false,
			Unsafe:   true,
			WSUnsafe: true,
			Host:     "localhost",
			Modules:  []string{"system", "author", "chain", "state"},
			WS:       false,
		},
	}
}
