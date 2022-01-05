// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/lib/common"
	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/pools"

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

func (t *Trie) string(tree gotree.Tree, curr Node, idx int) {
	switch c := curr.(type) {
	case *node.Branch:
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()

		_ = c.Encode(buffer)
		encoding := buffer.Bytes()

		var bstr string
		if len(encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d",
				idx, c, common.MustBlake2bHash(encoding), c.GetGeneration())
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), encoding, c.GetGeneration())
		}

		pools.EncodingBuffers.Put(buffer)

		sub := tree.Add(bstr)
		for i, child := range c.Children {
			if child != nil {
				t.string(sub, child, i)
			}
		}
	case *node.Leaf:
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()

		_ = c.Encode(buffer)

		encoding := buffer.Bytes()

		var bstr string
		if len(encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d",
				idx, c.String(), common.MustBlake2bHash(encoding), c.GetGeneration())
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), encoding, c.GetGeneration())
		}

		pools.EncodingBuffers.Put(buffer)

		tree.Add(bstr)
	default:
		return
	}
}
