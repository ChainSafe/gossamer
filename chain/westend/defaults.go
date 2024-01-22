// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westend

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/adrg/xdg"
)

var (
	// defaultBasePath is the default base directory path for westend node
	defaultBasePath = xdg.DataHome + "/gossamer/westend"
	// defaultChainSpec is the default chain specification path
	defaultChainSpec = "./chain/westend/chain-spec-raw.json"
)

// DefaultConfig returns a westend node configuration
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
