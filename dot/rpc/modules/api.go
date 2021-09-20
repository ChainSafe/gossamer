package modules

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// StorageAPI is the interface for the storage state
type StorageAPI interface {
	GetStorage(root *common.Hash, key []byte) ([]byte, error)
	GetStorageByBlockHash(bhash common.Hash, key []byte) ([]byte, error)
	Entries(root *common.Hash) (map[string][]byte, error)
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error)
	RegisterStorageObserver(observer state.Observer)
	UnregisterStorageObserver(observer state.Observer)
}

// BlockAPI is the interface for the block state
type BlockAPI interface {
	GetHeader(hash common.Hash) (*types.Header, error)
	BestBlockHash() common.Hash
	GetBlockByHash(hash common.Hash) (*types.Block, error)
	GetBlockHash(blockNumber *big.Int) (common.Hash, error)
	GetFinalisedHash(uint64, uint64) (common.Hash, error)
	GetHighestFinalisedHash() (common.Hash, error)
	HasJustification(hash common.Hash) (bool, error)
	GetJustification(hash common.Hash) ([]byte, error)
	RegisterImportedChannel(ch chan<- *types.Block) (byte, error)
	UnregisterImportedChannel(id byte)
	RegisterFinalizedChannel(ch chan<- *types.FinalisationInfo) (byte, error)
	UnregisterFinalisedChannel(id byte)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error)
	UnregisterRuntimeUpdatedChannel(id uint32) bool
}

// NetworkAPI interface for network state methods
type NetworkAPI interface {
	Health() common.Health
	NetworkState() common.NetworkState
	Peers() []common.PeerInfo
	NodeRoles() byte
	Stop() error
	Start() error
	IsStopped() bool
	HighestBlock() int64
	StartingBlock() int64
	AddReservedPeers(addrs ...string) error
	RemoveReservedPeers(addrs ...string) error
}

// BlockProducerAPI is the interface for BlockProducer methods
type BlockProducerAPI interface {
	Pause() error
	Resume() error
	EpochLength() uint64
	SlotDuration() uint64
}

// TransactionStateAPI ...
type TransactionStateAPI interface {
	AddToPool(*transaction.ValidTransaction) common.Hash
	Pop() *transaction.ValidTransaction
	Peek() *transaction.ValidTransaction
	Pending() []*transaction.ValidTransaction
}

// CoreAPI is the interface for the core methods
type CoreAPI interface {
	InsertKey(kp crypto.Keypair)
	HasKey(pubKeyStr string, keyType string) (bool, error)
	GetRuntimeVersion(bhash *common.Hash) (runtime.Version, error)
	HandleSubmittedExtrinsic(types.Extrinsic) error
	GetMetadata(bhash *common.Hash) ([]byte, error)
	QueryStorage(from, to common.Hash, keys ...string) (map[common.Hash]core.QueryKeyValueChanges, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	GetReadProofAt(block common.Hash, keys [][]byte) (common.Hash, [][]byte, error)
}

// RPCAPI is the interface for methods related to RPC service
type RPCAPI interface {
	Methods() []string
	BuildMethodNames(rcvr interface{}, name string)
}

// SystemAPI is the interface for handling system methods
type SystemAPI interface {
	SystemName() string
	SystemVersion() string
	Properties() map[string]interface{}
	ChainType() string
	ChainName() string
}

// BlockFinalityAPI is the interface for handling block finalisation methods
type BlockFinalityAPI interface {
	GetSetID() uint64
	GetRound() uint64
	GetVoters() grandpa.Voters
	PreVotes() []ed25519.PublicKeyBytes
	PreCommits() []ed25519.PublicKeyBytes
}

// RuntimeStorageAPI is the interface to interacts with the node storage
type RuntimeStorageAPI interface {
	SetLocal(k, v []byte) error
	SetPersistent(k, v []byte) error
	GetLocal(k []byte) ([]byte, error)
	GetPersistent(k []byte) ([]byte, error)
}
