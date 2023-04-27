// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package kusama

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

var (
	// defaultName Default node name
	defaultName = "Kusama"
	// defaultID Default chain ID
	defaultID = "ksmcc3"
	// defaultBasePath Default node base directory path
	defaultBasePath = "~/.gossamer/kusama"
	// defaultChainSpec is the default chain-spec json path
	defaultChainSpec = "./chain/kusama/genesis.json"
)

// DefaultConfig returns a kusama node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.BasePath = defaultBasePath
	config.ID = defaultID
	config.Name = defaultName
	config.ChainSpec = defaultChainSpec

	return config
}
