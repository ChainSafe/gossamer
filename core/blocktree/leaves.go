package blocktree

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
)

// leafMap provided quick lookup for existing leaves
type leafMap map[common.Hash]*node

func(ls leafMap) Replace(old, new *node) {
	delete(ls, old.hash)
	ls[new.hash] = new
}

func (ls leafMap) DeepestLeaf() *node {
	max := big.NewInt(-1)
	var dLeaf *node
	for _, n := range ls {
		if max.Cmp(n.depth) > 0 {
			max = n.depth
			dLeaf = n
		}
	}
	return dLeaf
}