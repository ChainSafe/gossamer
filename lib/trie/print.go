// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package trie

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/disiqueira/gotree"
)

// String returns the trie stringified through pre-order traversal
func (t *Trie) String() string {
	if t.root == nil {
		return "empty"
	}

	tree := gotree.New(fmt.Sprintf("Trie root=0x%x", t.root.getHash()))
	t.string(tree, t.root, 0)
	return fmt.Sprintf("\n%s", tree.Print())
}

func (t *Trie) string(tree gotree.Tree, curr node, idx int) {
	switch c := curr.(type) {
	case *branch:
		hasher := newHasher(false)
		defer hasher.returnToPool()
		c.encoding, _ = hasher.encode(c)
		var bstr string
		if len(c.encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.encoding), c.generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.encoding, c.generation)
		}
		sub := tree.Add(bstr)
		for i, child := range c.children {
			if child != nil {
				t.string(sub, child, i)
			}
		}
	case *leaf:
		hasher := newHasher(false)
		defer hasher.returnToPool()
		c.encoding, _ = hasher.encode(c)
		var bstr string
		if len(c.encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.encoding), c.generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.encoding, c.generation)
		}
		tree.Add(bstr)
	default:
		return
	}
}

// Print prints the trie through pre-order traversal
func (t *Trie) Print() {
	fmt.Println(t.String())
}
