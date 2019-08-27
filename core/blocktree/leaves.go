package blocktree

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
)

// leafMap provides quick lookup for existing leaves
type leafMap map[common.Hash]*node

// Replace deletes the old node from the map and inserts the new one
func(ls leafMap) Replace(old, new *node) {
	delete(ls, old.hash)
	ls[new.hash] = new
}

func (ls leafMap) DeepestLeaf() *node {
	max := big.NewInt(-1)
	var dLeaf *node
	for _, n := range ls {
		if max.Cmp(n.depth) < 0 {
			max = n.depth
			dLeaf = n
		}
	}
	return dLeaf
}