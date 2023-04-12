package config

import (
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/lib/common"
)

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
