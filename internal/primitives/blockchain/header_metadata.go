// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blockchain

import (
	"slices"
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
)

// / Get lowest common ancestor between two blocks in the tree.
// /
// / This implementation is efficient because our trees have very few and
// / small branches, and because of our current query pattern:
// / lca(best, final), lca(best + 1, final), lca(best + 2, final), etc.
// / The first call is O(h) but the others are O(1).
func LowestCommonAncestor[H runtime.Hash, N runtime.Number](
	backend HeaderMetadata[H, N], id1 H, id2 H,
) (HashNumber[H, N], error) {
	header1, err := backend.HeaderMetadata(id1)
	if err != nil {
		return HashNumber[H, N]{}, err
	}
	if header1.Parent == id2 {
		return HashNumber[H, N]{Hash: id2, Number: header1.Number - 1}, nil
	}

	header2, err := backend.HeaderMetadata(id2)
	if err != nil {
		return HashNumber[H, N]{}, err
	}
	if header2.Parent == id1 {
		return HashNumber[H, N]{Hash: id1, Number: header1.Number}, nil
	}

	origHeader1 := header1
	origHeader2 := header2

	// We move through ancestor links as much as possible, since ancestor >= parent.
	for header1.Number > header2.Number {
		ancestor1, err := backend.HeaderMetadata(header1.ancestor)
		if err != nil {
			return HashNumber[H, N]{}, err
		}

		if ancestor1.Number >= header2.Number {
			header1 = ancestor1
		} else {
			break
		}
	}
	for header1.Number < header2.Number {
		ancestor2, err := backend.HeaderMetadata(header2.ancestor)
		if err != nil {
			return HashNumber[H, N]{}, err
		}

		if ancestor2.Number > header1.Number {
			header2 = ancestor2
		} else {
			break
		}
	}

	// Then we move the remaining path using parent links.
	for header1.Hash != header2.Hash {
		if header1.Number > header2.Number {
			header1, err = backend.HeaderMetadata(header1.Parent)
			if err != nil {
				return HashNumber[H, N]{}, err
			}
		} else {
			header2, err = backend.HeaderMetadata(header2.Parent)
			if err != nil {
				return HashNumber[H, N]{}, err
			}
		}
	}

	// Update cached ancestor links.
	if origHeader1.Number > header1.Number {
		origHeader1.ancestor = header1.Hash
		backend.InsertHeaderMetadata(origHeader1.Hash, origHeader1)
	}
	if origHeader2.Number > header2.Number {
		origHeader2.ancestor = header1.Hash
		backend.InsertHeaderMetadata(origHeader2.Hash, origHeader2)
	}

	return HashNumber[H, N]{Hash: header1.Hash, Number: header1.Number}, nil
}

// / Compute a tree-route between two blocks. See tree-route docs for more details.
func NewTreeRoute[H runtime.Hash, N runtime.Number](
	backend HeaderMetadata[H, N], from, to H,
) (TreeRoute[H, N], error) {
	fromMeta, err := backend.HeaderMetadata(from)
	if err != nil {
		return TreeRoute[H, N]{}, err
	}
	toMeta, err := backend.HeaderMetadata(to)
	if err != nil {
		return TreeRoute[H, N]{}, err
	}

	var (
		fromBranch []HashNumber[H, N]
		toBranch   []HashNumber[H, N]
	)

	for toMeta.Number > fromMeta.Number {
		toBranch = append(toBranch, HashNumber[H, N]{Hash: toMeta.Hash, Number: toMeta.Number})

		toMeta, err = backend.HeaderMetadata(toMeta.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	for fromMeta.Number > toMeta.Number {
		fromBranch = append(fromBranch, HashNumber[H, N]{Hash: fromMeta.Hash, Number: fromMeta.Number})

		fromMeta, err = backend.HeaderMetadata(fromMeta.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	// numbers are equal now. walk backwards until the block is the same
	for toMeta.Hash != fromMeta.Hash {
		toBranch = append(toBranch, HashNumber[H, N]{Hash: toMeta.Hash, Number: toMeta.Number})
		toMeta, err = backend.HeaderMetadata(toMeta.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}

		fromBranch = append(fromBranch, HashNumber[H, N]{Hash: fromMeta.Hash, Number: fromMeta.Number})
		fromMeta, err = backend.HeaderMetadata(fromMeta.Parent)
		if err != nil {
			return TreeRoute[H, N]{}, err
		}
	}

	// add the pivot block. and append the reversed to-branch
	// (note that it's reverse order originals)
	pivot := uint(len(fromBranch))
	fromBranch = append(fromBranch, HashNumber[H, N]{Hash: toMeta.Hash, Number: toMeta.Number})
	slices.Reverse(toBranch)
	fromBranch = append(fromBranch, toBranch...)

	return TreeRoute[H, N]{Route: fromBranch, Pivot: pivot}, nil
}

// / Hash and number of a block.
type HashNumber[H runtime.Hash, N runtime.Number] struct {
	/// The number of the block.
	// pub number: NumberFor<Block>,
	Number N
	/// The hash of the block.
	// pub hash: Block::Hash,
	Hash H
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
type TreeRoute[H runtime.Hash, N runtime.Number] struct {
	// route: Vec<HashAndNumber<Block>>,
	Route []HashNumber[H, N]
	// pivot: usize,
	Pivot uint
}

// impl<Block: BlockT> TreeRoute<Block> {
// 	/// Creates a new `TreeRoute`.
// 	///
// 	/// To preserve the structure safety invariats it is required that `pivot < route.len()`.
// 	pub fn new(route: Vec<HashAndNumber<Block>>, pivot: usize) -> Result<Self, String> {
// 		if pivot < route.len() {
// 			Ok(TreeRoute { route, pivot })
// 		} else {
// 			Err(format!(
// 				"TreeRoute pivot ({}) should be less than route length ({})",
// 				pivot,
// 				route.len()
// 			))
// 		}
// 	}

// / Get a slice of all retracted blocks in reverse order (towards common ancestor).
func (tr TreeRoute[H, N]) Retracted() []HashNumber[H, N] {
	return tr.Route[:tr.Pivot]
}

// 	/// Convert into all retracted blocks in reverse order (towards common ancestor).
// 	pub fn into_retracted(mut self) -> Vec<HashAndNumber<Block>> {
// 		self.route.truncate(self.pivot);
// 		self.route
// 	}

// /// Get the common ancestor block. This might be one of the two blocks of the
// /// route.
//
//	pub fn common_block(&self) -> &HashAndNumber<Block> {
//		self.route.get(self.pivot).expect(
//			"tree-routes are computed between blocks; \
//			which are included in the route; \
//			thus it is never empty; qed",
//		)
//	}
func (tr TreeRoute[H, N]) CommonBlock() HashNumber[H, N] {
	return tr.Route[tr.Pivot]
}

// / Get a slice of enacted blocks (descendents of the common ancestor)
func (tr TreeRoute[H, N]) Enacted() []HashNumber[H, N] {
	return tr.Route[tr.Pivot+1:]
}

// 	/// Returns the last block.
// 	pub fn last(&self) -> Option<&HashAndNumber<Block>> {
// 		self.route.last()
// 	}
// }

// Handles header metadata: hash, number, parent hash, etc.
type HeaderMetadata[H, N any] interface {
	HeaderMetadata(hash H) (CachedHeaderMetadata[H, N], error)
	InsertHeaderMetadata(hash H, headerMetadata CachedHeaderMetadata[H, N])
	RemoveHeaderMetadata(hash H)
}

// Caches header metadata in an in-memory LRU cache.
type HeaderMetadataCache[H comparable, N any] struct {
	cache *lru.Cache[H, CachedHeaderMetadata[H, N]]
	sync.RWMutex
}

// NewHeaderMetadataCache is constructor for HeaderMetadataCache.
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

// HeaderMetadata returns the CachedHeaderMetadata for a given hash or `nil` if not found.
func (hmc *HeaderMetadataCache[H, N]) HeaderMetadata(hash H) *CachedHeaderMetadata[H, N] {
	hmc.RLock()
	defer hmc.RUnlock()
	val, ok := hmc.cache.Get(hash)
	if !ok {
		return nil
	}
	return &val
}

// InsertHeaderMetadata inserts a supplied `metadata` for a `hash`.
func (hmc *HeaderMetadataCache[H, N]) InsertHeaderMetadata(hash H, metadata CachedHeaderMetadata[H, N]) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Add(hash, metadata)
}

// RemoveHeaderMetadata removes the `metadata` for a `hash`.
func (hmc *HeaderMetadataCache[H, N]) RemoveHeaderMetadata(hash H) {
	hmc.Lock()
	defer hmc.Unlock()
	hmc.cache.Remove(hash)
}

// CachedHeaderMeatadata is used to efficiently traverse the tree.
type CachedHeaderMetadata[H, N any] struct {
	// Hash of the header.
	Hash H
	// Block number.
	Number N
	// Hash of parent header.
	Parent H
	// Block state root.
	StateRoot H
	// Hash of an ancestor header. Used to jump through the tree.
	ancestor H
}

// NewCachedHeaderMetadata is constructor for CachedHeaderMetadata
func NewCachedHeaderMetadata[H runtime.Hash, N runtime.Number](header runtime.Header[N, H]) CachedHeaderMetadata[H, N] {
	return CachedHeaderMetadata[H, N]{
		Hash:      header.Hash(),
		Number:    header.Number(),
		Parent:    header.ParentHash(),
		StateRoot: header.StateRoot(),
		ancestor:  header.ParentHash(),
	}
}
