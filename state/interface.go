package state

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/p2p"
)

// Read only
type ROStorageApi interface {
	ExistsStorage(key []byte) bool
	GetStorage(key []byte) []byte
	StorageRoot() common.Hash
	EnumeratedTrieRoot(values [][]byte)
	//TODO: add child storage funcs
}

type StorageApi interface {
	ROStorageApi
	SetStorage(key []byte, value []byte)
	ClearPrefix(prefix []byte)
	ClearStorage(key []byte)
	// TODO: child storage funcs
}

// Read only
type ROBlockApi interface {
	GetHeader(hash common.Hash) types.BlockHeader
	GetBlockData(hash common.Hash) types.BlockData
	GetLatestBlock() types.BlockHeader
	GetBlockByHash(hash common.Hash) types.Block
	GetBlockByNumber(n *big.Int) types.Block
}

type BlockApi interface {
	ROBlockApi
	SetHeader(header types.BlockHeader)
	SetBlockData(hash common.Hash, header types.BlockHeader)
}

type MessageApi interface {
	PushMessage(msg p2p.Message)
}

type PeerApi interface {
	//GetEventStream() chan<- p2p.Event
	Peers() []string
	State() string
}

type NetworkApi interface {
	// Network
	PeerCount() int
	Status() string
}
