package modules

import (
	"github.com/ChainSafe/gossamer/common"
)

type StorageAPI interface{}

type BlockAPI interface{}

// NetworkApi interface for network state methods
type NetworkAPI interface {
	Health() *common.Health
	NetworkState() *common.NetworkState
	Peers() []common.PeerInfo
}

type CoreAPI interface{}
