package api

import (
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type KeyValue struct {
	Key   []byte
	Value []byte
}

// / Provides access to an auxiliary database.
// /
// / This is a simple global database not aware of forks. Can be used for storing auxiliary
// / information like total block weight/difficulty for fork resolution purposes as a common use
// / case.
type AuxStore interface {
	// Insert auxiliary data into key-value store.
	// Deletions occur after insertions.
	InsertAux(insert []KeyValue, delete [][]byte) error

	// Query auxiliary data from key-value store.
	GetAux(key []byte) (*[]byte, error)
}

// / Client backend.
// /
// / Manages the data layer.
// /
// / # State Pruning
// /
// / While an object from `state_at` is alive, the state
// / should not be pruned. The backend should internally reference-count
// / its state objects.
// /
// / The same applies for live `BlockImportOperation`s: while an import operation building on a
// / parent `P` is alive, the state for `P` should not be pruned.
// /
// / # Block Pruning
// /
// / Users can pin blocks in memory by calling `pin_block`. When
// / a block would be pruned, its value is kept in an in-memory cache
// / until it is unpinned via `unpin_block`.
// /
// / While a block is pinned, its state is also preserved.
// /
// / The backend should internally reference count the number of pin / unpin calls.
type Backend[H runtime.Hash, N runtime.Number, Hasher runtime.Hasher[H]] interface {
	AuxStore

	// Returns reference to blockchain backend.
	Blockchain() blockchain.Backend[H, N]
}
