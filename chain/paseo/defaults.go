// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package paseo

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// DefaultBasePath default node base directory path
	DefaultBasePath = xdg.DataHome + "/gossamer/paseo"
	// DefaultChainSpec is the default chain spec configuration path
	DefaultChainSpec = "./chain/paseo/chain-spec-raw.json"
)

// DefaultConfig returns a paseo node configuration
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
