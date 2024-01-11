// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// defaultName Default node name
	defaultName = "Kusama"
	// defaultID Default chain ID
	defaultID = "ksmcc3"
	// defaultConfigDir is the default config directory path
	defaultConfigDir = xdg.ConfigHome + "/kusama/config"
	// defaultDataPath is the default data directory path
	defaultDataPath = xdg.DataHome + "/kusama/data"
	// defaultChainSpec is the default chain-spec json path
	defaultChainSpec = "./chain/kusama/chain-spec-raw.json"
)

// DefaultConfig returns a kusama node configuration
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
