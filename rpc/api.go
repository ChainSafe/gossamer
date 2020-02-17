package rpc

import (
	"github.com/ChainSafe/gossamer/common"
)

type StorageApi interface{}

type BlockApi interface{}

// NetworkApi interface for network state methods
type NetworkApi interface {
	Health() *common.Health
	NetworkState() *common.NetworkState
	Peers() []common.PeerInfo
}

type CoreApi interface{}
