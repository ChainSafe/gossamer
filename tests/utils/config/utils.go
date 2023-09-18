// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ParseNetworkRole converts a common.NetworkRole to a string representation.
func ParseNetworkRole(r common.NetworkRole) string {
	switch r {
	case common.NoNetworkRole:
		return cfg.NoNetworkRole.String()
	case common.FullNodeRole:
		return cfg.FullNode.String()
	case common.LightClientRole:
		return cfg.LightNode.String()
	case common.AuthorityRole:
		return cfg.AuthorityNode.String()
	default:
		return "Unknown"
	}
}
