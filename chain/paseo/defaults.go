// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package paseo

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// defaultName Default node name
	defaultName = "Paseo"
	// defaultID Default chain ID
	defaultID = "paseo"
	// defaultConfigDir is the default config directory path
	defaultConfigDir = xdg.ConfigHome + "/gossamer/paseo"
	// defaultDataPath is the default data directory path
	defaultDataPath = xdg.DataHome + "/gossamer/paseo"
	// defaultChainSpec is the default chain spec configuration path
	defaultChainSpec = "./chain/paseo/chain-spec-raw.json"
)

// DefaultConfig returns a paseo node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.ConfigDir = defaultConfigDir
	config.DataDir = defaultDataPath
	config.ID = defaultID
	config.Name = defaultName
	config.ChainSpec = defaultChainSpec
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.Core.Role = 1
	config.Network.NoMDNS = false

	return config
}
