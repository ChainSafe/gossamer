package modules

import (
	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
)

// StorageAPI is the interface for the storage state
type StorageAPI interface{}

// BlockAPI is the interface for the block state
type BlockAPI interface{}

// NetworkAPI interface for network state methods
type NetworkAPI interface {
	Health() *common.Health
	NetworkState() *common.NetworkState
	Peers() []common.PeerInfo
}

// TransactionQueueAPI ...
type TransactionQueueAPI interface {
	Push(*tx.ValidTransaction)
	Pop() *tx.ValidTransaction
	Peek() *tx.ValidTransaction
	Pending() []*tx.ValidTransaction
}

// CoreAPI is the interface for the core methods
type CoreAPI interface{}
