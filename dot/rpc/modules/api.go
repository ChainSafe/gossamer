package modules

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// StorageAPI is the interface for the storage state
type StorageAPI interface {
	GetStorage(key []byte) ([]byte, error)
	Entries() map[string][]byte
}

// BlockAPI is the interface for the block state
type BlockAPI interface {
	GetHeader(hash common.Hash) (*types.Header, error)
	HighestBlockHash() common.Hash
	GetBlockByHash(hash common.Hash) (*types.Block, error)
	GetBlockHash(blockNumber *big.Int) (*common.Hash, error)
	SetBlockAddedChannel(chan<- *types.Block, <-chan struct{})
	GetFinalizedHash(uint64) (common.Hash, error)
}

// NetworkAPI interface for network state methods
type NetworkAPI interface {
	Health() common.Health
	NetworkState() common.NetworkState
	Peers() []common.PeerInfo
	NodeRoles() byte
}

// TransactionQueueAPI ...
type TransactionQueueAPI interface {
	Push(*transaction.ValidTransaction) (common.Hash, error)
	Pop() *transaction.ValidTransaction
	Peek() *transaction.ValidTransaction
	Pending() []*transaction.ValidTransaction
}

// CoreAPI is the interface for the core methods
type CoreAPI interface {
	InsertKey(kp crypto.Keypair)
	HasKey(pubKeyStr string, keyType string) (bool, error)
	GetRuntimeVersion() (*runtime.VersionAPI, error)
	IsBlockProducer() bool
	HandleSubmittedExtrinsic(types.Extrinsic) error
	GetMetadata() ([]byte, error)
}

// RPCAPI is the interface for methods related to RPC service
type RPCAPI interface {
	Methods() []string
	BuildMethodNames(rcvr interface{}, name string)
}

// RuntimeAPI is the interface for runtime methods
type RuntimeAPI interface {
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
}

// SystemAPI is the interface for handling system methods
type SystemAPI interface {
	SystemName() string
	SystemVersion() string
	NodeName() string
	Properties() map[string]interface{}
}
