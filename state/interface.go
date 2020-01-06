package state

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/p2p"
)

// Read only
type ROStorageApi interface {
	ExistsStorage(key []byte) (bool, error)
	GetStorage(key []byte) ([]byte, error)
	StorageRoot() (common.Hash, error)
	EnumeratedTrieRoot(values [][]byte)
	//TODO: add child storage funcs
}

type StorageApi interface {
	ROStorageApi
	SetStorage(key []byte, value []byte) error
	ClearPrefix(prefix []byte)
	ClearStorage(key []byte) error
	// TODO: child storage funcs
}

// Read only
type ROBlockApi interface {
	GetHeader(hash common.Hash) (types.BlockHeader, error)
	GetBlockData(hash common.Hash) (types.BlockData, error)
	GetLatestBlock() types.BlockHeader
	GetBlockByHash(hash common.Hash) (types.Block, error)
	GetBlockByNumber(n *big.Int) (types.Block, error)
}

type BlockApi interface {
	ROBlockApi
	SetHeader(header types.BlockHeader) error
	SetBlockData(hash common.Hash, header types.BlockHeader) error
}

type MessageApi interface {
	PushMessage(msg p2p.Message)
}

type PeerApi interface {
	//GetEventStream() chan<- p2p.Event
	// Peers() []PeerInfo
	State() string
}

type NetworkApi interface {
	// Network
	PeerCount() int
	Peers() []string
	Status() string
}
