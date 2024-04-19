package blockchain

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

// / Blockchain database header backend. Does not perform any validation.
type HeaderBackend[Hash runtime.Hash, N runtime.Number] interface {
	/// Get block header. Returns `None` if block is not found.
	Header(hash Hash) (*runtime.Header[N, Hash], error)

	/// Get blockchain info.
	Info() Info[Hash, N]

	/// Get block status.
	Status(hash Hash) (BlockStatus, error)

	/// Get block number by hash. Returns `None` if the header is not in the chain.
	Number(hash Hash) (*N, error)

	/// Get block hash by number. Returns `None` if the header is not in the chain.
	Hash(number N) (*Hash, error)

	/// Convert an arbitrary block ID into a block hash.
	BlockHashFromID(id generic.BlockID) (*Hash, error)

	/// Convert an arbitrary block ID into a block hash.
	BlockNumberFromID(id generic.BlockID) (*N, error)

	/// Get block header. Returns `UnknownBlock` error if block is not found.
	ExpectHeader(hash Hash) (runtime.Header[N, Hash], error)

	/// Convert an arbitrary block ID into a block number. Returns `UnknownBlock` error if block is
	/// not found.
	ExpectBlockNumberFromID(id generic.BlockID) (N, error)

	/// Convert an arbitrary block ID into a block hash. Returns `UnknownBlock` error if block is
	/// not found.
	ExpectBlockHashFromID(id generic.BlockID) (Hash, error)
}

// / Blockchain database backend. Does not perform any validation.
type Backend[Hash runtime.Hash, N runtime.Number] interface {
	HeaderBackend[Hash, N]
	// HeaderMetaData[Hash, N]

	/// Get block body. Returns `None` if block is not found.
	Body(hash Hash) (*[]runtime.Extrinsic, error)
	/// Get block justifications. Returns `None` if no justification exists.
	Justifications(hash Hash) (*runtime.Justifications, error)
	/// Get last finalized block hash.
	LastFinalized() (Hash, error)

	/// Returns hashes of all blocks that are leaves of the block tree.
	/// in other words, that have no children, are chain heads.
	/// Results must be ordered best (longest, highest) chain first.
	Leaves() ([]Hash, error)

	/// Returns displaced leaves after the given block would be finalized.
	///
	/// The returned leaves do not contain the leaves from the same height as `block_number`.
	DisplacedLeavesAfterFinalizing(blockNumber N) ([]Hash, error)

	/// Return hashes of all blocks that are children of the block with `parent_hash`.
	Children(parentHash Hash) ([]Hash, error)

	/// Get the most recent block hash of the longest chain that contains
	/// a block with the given `base_hash`.
	///
	/// The search space is always limited to blocks which are in the finalized
	/// chain or descendents of it.
	///
	/// Returns `Ok(None)` if `base_hash` is not found in search space.
	// TODO: document time complexity of this, see [#1444](https://github.com/paritytech/substrate/issues/1444)
	LongestContaining(baseHash Hash, importLock *sync.RWMutex) (*Hash, error)

	/// Get single indexed transaction by content hash. Note that this will only fetch transactions
	/// that are indexed by the runtime with `storage_index_transaction`.
	IndexedTransaction(hash Hash) (*[]byte, error)

	/// Check if indexed transaction exists.
	HasIndexedTransaction(hash Hash) (bool, error)

	BlockIndexedBody(hash Hash) (*[][]byte, error)
}

// / Blockchain info
type Info[H, N any] struct {
	/// Best block hash.
	BestHash H
	/// Best block number.
	BestNumber N
	/// Genesis block hash.
	GenesisHash H
	/// The head of the finalized chain.
	FinalizedHash H
	/// Last finalized block number.
	FinalizedNumber N
	/// Last finalized state.
	FinalizedState *struct {
		Hash   H
		Number N
	}
	/// Number of concurrent leave forks.
	NumberLeaves uint
	/// Missing blocks after warp sync. (start, end).
	BlockGap *[2]N
}

// / Block status.
type BlockStatus uint

const (
	/// Already in the blockchain.
	BlockStatusInChain BlockStatus = iota
	/// Not in the queue or the blockchain.
	BlockStatusUnknown
)
