package blockchain

import (
	"slices"
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
)

// / Hash and number of a block.
type HashNumber[H any, N any] struct {
	/// The hash of the block.
	Hash H
	/// The number of the block.
	Number N
}

// / A tree-route from one block to another in the chain.
// /
// / All blocks prior to the pivot in the deque is the reverse-order unique ancestry
// / of the first block, the block at the pivot index is the common ancestor,
// / and all blocks after the pivot is the ancestry of the second block, in
// / order.
// /
// / The ancestry sets will include the given blocks, and thus the tree-route is
// / never empty.
// /
// / ```text
// / Tree route from R1 to E2. Retracted is [R1, R2, R3], Common is C, enacted [E1, E2]
// /   <- R3 <- R2 <- R1
// /  /
// / C
// /  \-> E1 -> E2
// / ```
// /
// / ```text
// / Tree route from C to E2. Retracted empty. Common is C, enacted [E1, E2]
// / C -> E1 -> E2
// / ```
// #[derive(Debug, Clone)]
//
//	pub struct TreeRoute<Block: BlockT> {
//		route: Vec<HashAndNumber<Block>>,
//		pivot: usize,
//	}
type TreeRoute[H, N any] struct {
	route []HashNumber[H, N]
	pivot uint
}

// / Creates a new `TreeRoute`.
// /
// / To preserve the structure safety invariats it is required that `pivot < route.len()`.
func NewTreeRoute[H runtime.Hash, N runtime.Number](backend HeaderMetaData[H, N], from H, to H) (TreeRoute[H, N], error) {
	fromData, err := backend.HeaderMetadata(from)
	if err != nil {
		return TreeRoute[H, N]{}, err
	}
	toData, err := backend.HeaderMetadata(to)
	if err != nil {
		return TreeRoute[H, N]{}, err
	}

	var (
		fromBranch []HashNumber[H, N]
		toBranch   []HashNumber[H, N]
	)

	for toData.Number > fromData.Number {
		toBranch = append(toBranch, HashNumber[H, N]{toData.Hash, toData.Number})
		toData, err = backend.HeaderMetadata(toData.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	for fromData.Number > toData.Number {
		fromBranch = append(fromBranch, HashNumber[H, N]{fromData.Hash, fromData.Number})
		fromData, err = backend.HeaderMetadata(fromData.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	// numbers are equal now. walk backwards until the block is the same
	for toData.Hash != fromData.Hash {
		toBranch = append(toBranch, HashNumber[H, N]{toData.Hash, toData.Number})
		toData, err = backend.HeaderMetadata(toData.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}

		fromBranch = append(fromBranch, HashNumber[H, N]{fromData.Hash, fromData.Number})
		fromData, err = backend.HeaderMetadata(fromData.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	// add the pivot block. and append the reversed to-branch
	// (note that it's reverse order originals)
	pivot := uint(len(fromBranch))
	fromBranch = append(fromBranch, HashNumber[H, N]{toData.Hash, toData.Number})
	slices.Reverse(toBranch)
	fromBranch = append(fromBranch, toBranch...)

	return TreeRoute[H, N]{
		route: fromBranch,
		pivot: pivot,
	}, nil
}

// / Get a slice of all retracted blocks in reverse order (towards common ancestor).
func (tr TreeRoute[H, N]) Retratcted() []HashNumber[H, N] {
	return tr.route[0:tr.pivot]
}

// / Get the common ancestor block. This might be one of the two blocks of the
// / route.
func (tr TreeRoute[H, N]) CommonBlock() HashNumber[H, N] {
	// self.route.get(self.pivot).expect(
	// 		"tree-routes are computed between blocks; \
	// 		which are included in the route; \
	// 		thus it is never empty; qed",
	// 	)
	return tr.route[tr.pivot]
}

// / Get a slice of enacted blocks (descendents of the common ancestor)
func (tr TreeRoute[H, N]) Enacted() []HashNumber[H, N] {
	return tr.route[tr.pivot+1:]
}

func (tr TreeRoute[H, N]) Last() *HashNumber[H, N] {
	return &tr.route[len(tr.route)-1]
}

// / Handles header metadata: hash, number, parent hash, etc.
// pub trait HeaderMetadata<Block: BlockT> {
type HeaderMetaData[H, N any] interface {
	// fn header_metadata(
	// 	&self,
	// 	hash: Block::Hash,
	// ) -> Result<CachedHeaderMetadata<Block>, Self::Error>;
	HeaderMetadata(hash H) (CachedHeaderMetadata[H, N], error)
	// fn insert_header_metadata(
	// 	&self,
	// 	hash: Block::Hash,
	// 	header_metadata: CachedHeaderMetadata<Block>,
	// );
	InsertHeaderMetadata(hash H, headerMetadata CachedHeaderMetadata[H, N])
	// fn remove_header_metadata(&self, hash: Block::Hash);
	RemoveHeaderMetadata(hash H)
}

// / Caches header metadata in an in-memory LRU cache.
type HeaderMetadataCache[H comparable, N any] struct {
	cache *lru.Cache[H, CachedHeaderMetadata[H, N]]
	sync.RWMutex
}

func NewHeaderMetadataCache[H comparable, N any](capacity ...uint32) HeaderMetadataCache[H, N] {
	var cap int = 5000
	if len(capacity) > 0 && capacity[0] > 0 {
		cap = int(capacity[0])
	}
	cache, err := lru.New[H, CachedHeaderMetadata[H, N]](cap)
	if err != nil {
		panic(err)
	}
	return HeaderMetadataCache[H, N]{
		cache: cache,
	}
}

func (hmc *HeaderMetadataCache[H, N]) HeaderMetadata(hash H) *CachedHeaderMetadata[H, N] {
	hmc.RLock()
	defer hmc.RUnlock()
	val, ok := hmc.cache.Get(hash)
	if !ok {
		return nil
	}
	return &val
}

func (hmc *HeaderMetadataCache[H, N]) InsertHeaderMetadata(hash H, metadata CachedHeaderMetadata[H, N]) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Add(hash, metadata)
	return
}

func (hmc *HeaderMetadataCache[H, N]) RemoveHeaderMetadata(hash H) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Remove(hash)
	return
}

// / Cached header metadata. Used to efficiently traverse the tree.
// pub struct CachedHeaderMetadata<Block: BlockT> {
type CachedHeaderMetadata[H, N any] struct {
	/// Hash of the header.
	// pub hash: Block::Hash,
	Hash H
	/// Block number.
	// pub number: NumberFor<Block>,
	Number N
	/// Hash of parent header.
	// pub parent: Block::Hash,
	Parent H
	/// Block state root.
	// pub state_root: Block::Hash,
	StateRoot H
	/// Hash of an ancestor header. Used to jump through the tree.
	// ancestor: Block::Hash,
	ancestor H
}

func NewCachedHeaderMetadata[H runtime.Hash, N runtime.Number](header runtime.Header[N, H]) CachedHeaderMetadata[H, N] {
	return CachedHeaderMetadata[H, N]{
		Hash:      header.Hash(),
		Number:    header.Number(),
		Parent:    header.ParentHash(),
		StateRoot: header.StateRoot(),
		ancestor:  header.ParentHash(),
	}
}
