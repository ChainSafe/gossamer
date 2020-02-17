package modules

import (
	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
)

type StorageApi interface{}

type BlockApi interface{}

// NetworkApi interface for network state methods
type NetworkApi interface {
	Health() *common.Health
	NetworkState() *common.NetworkState
	Peers() []common.PeerInfo
}

type CoreApi interface {
	PushToTxQueue(*tx.ValidTransaction)
}
