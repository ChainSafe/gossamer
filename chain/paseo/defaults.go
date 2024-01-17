// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package paseo

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// defaultBasePath default node base directory path
	defaultBasePath = xdg.DataHome + "/gossamer/paseo"
	// defaultChainSpec is the default chain spec configuration path
	defaultChainSpec = "./chain/paseo/chain-spec-raw.json"
)

// DefaultConfig returns a paseo node configuration
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
