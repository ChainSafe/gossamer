package sync

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

type DisjointBlockSet interface {
	addHashAndNumber(common.Hash, *big.Int)
	addHeader(*types.Header)
	addBlock(*types.Block)
	updateBlock(common.Hash, *types.Block)
	removeBlock(common.Hash)
}

type disjointBlockSet struct {
	set map[common.Hash]*types.Block
}

func newDisjointBlockSet() *disjointBlockSet {
	return &disjointBlockSet{}
}

func (s *disjointBlockSet) addHashAndNumber(common.Hash, *big.Int) {}
func (s *disjointBlockSet) addHeader(*types.Header)                {}
func (s *disjointBlockSet) addBlock(*types.Block)                  {}
func (s *disjointBlockSet) updateBlock(common.Hash, *types.Block)  {}
func (s *disjointBlockSet) removeBlock(common.Hash)                {}
