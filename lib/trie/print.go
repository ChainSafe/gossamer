// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/branch"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie/pools"

	"github.com/disiqueira/gotree"
)

// String returns the trie stringified through pre-order traversal
func (t *Trie) String() string {
	if t.root == nil {
		return "empty"
	}

	tree := gotree.New(fmt.Sprintf("Trie root=0x%x", t.root.GetHash()))
	t.string(tree, t.root, 0)
	return fmt.Sprintf("\n%s", tree.Print())
}

func (t *Trie) string(tree gotree.Tree, curr node.Node, idx int) {
	switch c := curr.(type) {
	case *branch.Branch:
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()

		_ = c.Encode(buffer)
		c.Encoding = buffer.Bytes()

		var bstr string
		if len(c.Encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.Encoding), c.Generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.Encoding, c.Generation)
		}

		pools.EncodingBuffers.Put(buffer)

		sub := tree.Add(bstr)
		for i, child := range c.Children {
			if child != nil {
				t.string(sub, child, i)
			}
		}
	case *leaf.Leaf:
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()

		_ = c.Encode(buffer)

		// TODO lock or use methods on leaf to set the encoding bytes.
		// Right now this is only used for debugging so no need to lock
		c.Encoding = buffer.Bytes()

		var bstr string
		if len(c.Encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.Encoding), c.Generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.Encoding, c.Generation)
		}

		pools.EncodingBuffers.Put(buffer)

		tree.Add(bstr)
	default:
		return
	}
}
