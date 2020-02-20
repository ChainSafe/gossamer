package modules

import (
	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
)

// StorageAPI ...
type StorageAPI interface{}

// BlockAPI ...
type BlockAPI interface{}

// NetworkAPI interface for network state methods
type NetworkAPI interface {
	Health() *common.Health
	NetworkState() *common.NetworkState
	Peers() []common.PeerInfo
}

// CoreAPI ...
type CoreAPI interface {
	PushToTxQueue(*tx.ValidTransaction)
}
