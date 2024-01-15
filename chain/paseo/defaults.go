// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package paseo

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

var (
	// defaultName Default node name
	defaultName = "Paseo"
	// defaultID Default chain ID
	defaultID = "paseo"
	// defaultBasePath Default node base directory path
	defaultBasePath = "~/.gossamer/paseo"
	// defaultChainSpec is the default chain spec configuration path
	defaultChainSpec = "./chain/paseo/chain-spec-raw.json"
)

// DefaultConfig returns a paseo node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.BasePath = defaultBasePath
	config.ID = defaultID
	config.Name = defaultName
	config.ChainSpec = defaultChainSpec
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	config.Core.Role = 1
	config.Network.NoMDNS = false

	return config
}
