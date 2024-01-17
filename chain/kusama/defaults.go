// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/adrg/xdg"
)

var (
	// defaultBasePath default node base directory path
	defaultBasePath = xdg.DataHome + "/gossamer/kusama"
	// defaultChainSpec is the default chain-spec json path
	defaultChainSpec = "./chain/kusama/chain-spec-raw.json"
)

// DefaultConfig returns a kusama node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.BasePath = defaultBasePath
	config.ChainSpec = defaultChainSpec
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.Core.Role = 1
	config.Network.NoMDNS = false

	return config
}
