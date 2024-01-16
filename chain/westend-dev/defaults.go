// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westenddev

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// defaultName is the default node name
	defaultName = "Westend"
	// defaultID is the default node ID
	defaultID = "westend_dev"
	// defaultConfigDir is the default config directory path
	defaultConfigDir = xdg.ConfigHome + "/gossamer/westend-dev"
	// defaultDataPath is the default data directory path
	defaultDataPath = xdg.DataHome + "/gossamer/westend-dev"
	// defaultChainSpec is the default chain spec for the westend dev node
	defaultChainSpec = "./chain/westend-dev/westend-dev-spec-raw.json"
)

// DefaultConfig returns a westend dev node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.ConfigDir = defaultConfigDir
	config.DataDir = defaultDataPath
	config.ID = defaultID
	config.Name = defaultName
	config.ChainSpec = defaultChainSpec
	config.RPC.RPCExternal = true
	config.RPC.UnsafeRPC = true
	config.RPC.WSExternal = true
	config.RPC.UnsafeWSExternal = true

	return config
}
