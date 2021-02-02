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

package trie

import (
	"fmt"

	"github.com/disiqueira/gotree"
)

// String returns the trie stringified through pre-order traversal
func (t *Trie) String() string {
	if t.root == nil {
		return "empty"
	}

	tree := gotree.New("Trie")
	t.string(tree, t.root, 0)
	return fmt.Sprintf("\n%s", tree.Print())
}

func (t *Trie) string(tree gotree.Tree, curr node, idx int) {
	switch c := curr.(type) {
	case *branch:
		sub := tree.Add(fmt.Sprintf("idx=%d %s enc=%x", idx, c.String(), c.encoding))
		for i, child := range c.children {
			if child != nil {
				t.string(sub, child, i)
			}
		}
	case *leaf:
		tree.Add(fmt.Sprintf("idx=%d %s enc=%x", idx, c.String(), c.encoding))
	default:
		return
	}
}

// StringWithEncoding returns the trie stringified as well as the encoding of each node
func (t *Trie) StringWithEncoding() string {
	return t.String()
}

// Print prints the trie through pre-order traversal
func (t *Trie) Print() {
	fmt.Println(t.String())
}
