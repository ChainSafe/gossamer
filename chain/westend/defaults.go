// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westend

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

var (
	// defaultName Default node name
	defaultName = "Westend"
	// defaultID Default chain ID
	defaultID = "westend2"
	// defaultBasePath Default node base directory path
	defaultBasePath = "~/.gossamer/westend"
	// defaultChainSpec is the default chain specification path
	defaultChainSpec = "./chain/westend/genesis.json"
)

// DefaultConfig returns a westend node configuration
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

	config.Log.Sync = "trace"
	config.Pprof.Enabled = true
	config.Pprof.ListeningAddress = "0.0.0.0:6060"

	return config
}
