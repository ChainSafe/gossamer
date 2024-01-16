// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/adrg/xdg"
)

var (
	// DefaultBasePath default node base directory path
	DefaultBasePath = xdg.DataHome + "/gossamer/kusama"
	// DefaultChainSpec is the default chain-spec json path
	DefaultChainSpec = "./chain/kusama/chain-spec-raw.json"
)

// DefaultConfig returns a kusama node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.BasePath = DefaultBasePath
	config.ChainSpec = DefaultChainSpec
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.Core.Role = 1
	config.Network.NoMDNS = false

	return config
}
