// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blockchain

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

// HeaderBackend is the blockchain database header backend. Does not perform any validation.
type HeaderBackend[Hash runtime.Hash, N runtime.Number, Header runtime.Header[N, Hash]] interface {
	// Header returns the block header. Returns nil if block is not found.
	Header(hash Hash) (*Header, error)

	// Info returns blockchain [Info].
	Info() Info[Hash, N]

	// Status returns [BlockStatus].
	Status(hash Hash) (BlockStatus, error)

	// Number returns block number by hash. Returns nil if the header is not in the chain.
	Number(hash Hash) (*N, error)

	// Hash returns block hash by number. Returns nil if the header is not in the chain.
	Hash(number N) (*Hash, error)

	// BlockHashFromID converts an arbitrary block ID into a block hash.
	BlockHashFromID(id generic.BlockID) (*Hash, error)

	// BlockNumberFromID converts an arbitrary block ID into a block hash.
	BlockNumberFromID(id generic.BlockID) (*N, error)
}

// Backend is a blockchain database backend. Does not perform any validation.
type Backend[Hash runtime.Hash, N runtime.Number, Header runtime.Header[N, Hash], E runtime.Extrinsic] interface {
	HeaderBackend[Hash, N, Header]
	HeaderMetadata[Hash, N]

	// Body returns block body. Returns nil if block is not found.
	Body(hash Hash) ([]E, error)
	// Justifications returns block justifications. Returns nil if no justification exists.
	Justifications(hash Hash) (runtime.Justifications, error)
	// LastFinalized returns last finalized block hash.
	LastFinalized() (Hash, error)

	// Leaves returns hashes of all blocks that are leaves of the block tree.
	// in other words, that have no children, are chain heads.
	// Results must be ordered best (longest, highest) chain first.
	Leaves() ([]Hash, error)

	// DisplacedLeavesAfterFinalizing returns displaced leaves after the given block would be finalized.
	// The returned leaves do not contain the leaves from the same height as blockNumber.
	DisplacedLeavesAfterFinalizing(blockNumber N) ([]Hash, error)

	// Children returns hashes of all blocks that are children of the block with parentHash.
	Children(parentHash Hash) ([]Hash, error)

	// LongestContaining gets the most recent block hash of the longest chain that contains
	// a block with the given baseHash.

	// The search space is always limited to blocks which are in the finalized
	// chain or descendents of it.
	//
	// Returns nil if basehash is not found in search space.
	LongestContaining(baseHash Hash, importLock *sync.RWMutex) (*Hash, error)

	// IndexedTransaction returns single indexed transaction by content hash. Note that this will only fetch transactions
	// that are indexed by the runtime with storage_index_transaction.
	IndexedTransaction(hash Hash) ([]byte, error)

	// HasIndexedTransaction checks if indexed transaction exists.
	HasIndexedTransaction(hash Hash) (bool, error)

	// BlockIndexedBody will return the indexed body if it exists.
	BlockIndexedBody(hash Hash) ([][]byte, error)
}

// Info is the blockchain info
type Info[H, N any] struct {
	BestHash        H         // Best block hash.
	BestNumber      N         // Best block number.
	GenesisHash     H         // Genesis block hash.
	FinalizedHash   H         // The head of the finalized chain.
	FinalizedNumber N         // Last finalized block number.
	FinalizedState  *struct { // Last finalized state.
		Hash   H
		Number N
	}
	NumberLeaves uint  // Number of concurrent leave forks.
	BlockGap     *[2]N // Missing blocks after warp sync. (start, end).
}

// BlockStatus is block status.
type BlockStatus uint

const (
	// BlockStatusInChain represents block is already in the blockchain.
	BlockStatusInChain BlockStatus = iota
	// BlockStatusUnknown represents block is not in the queue or the blockchain.
	BlockStatusUnknown
)
