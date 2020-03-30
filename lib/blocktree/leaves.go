// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package blocktree

import (
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

// leafMap provides quick lookup for existing leaves
type leafMap struct {
	smap *sync.Map // map[common.Hash]*node
}

func NewLeafMap() *leafMap {
	return &leafMap{
		smap: &sync.Map{},
	}
}

func (ls *leafMap) store(key Hash, value *node) {
	ls.smap.Store(key, value)
}

func (ls *leafMap) load(key Hash) (*node, error) {
	v, ok := ls.smap.Lead(key)
	if !ok {
		return errors.New("Key not found")
	}

	return v.(*node), nil
}

// Replace deletes the old node from the map and inserts the new one
func (ls *leafMap) replace(old, new *node) {
	ls.Delete(old.hash)
	ls.Store(new.hash, new)
}

// DeepestLeaf searches the stored leaves to the find the one with the greatest depth.
// If there are two leaves with the same depth, choose the one with the earliest arrival time.
func (ls *leafMap) deepestLeaf() *node {
	max := big.NewInt(-1)

	ls.Range(func(hash, node) bool {

	}())

	var dLeaf *node
	for _, n := range ls {
		if max.Cmp(n.depth) < 0 {
			max = n.depth
			dLeaf = n
		} else if max.Cmp(n.depth) == 0 && n.arrivalTime > dLeaf.arrivalTime {
			dLeaf = n
		}
	}
	return dLeaf
}
