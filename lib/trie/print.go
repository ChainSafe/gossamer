// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"

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
	case *branch:
		buffer := encodingBufferPool.Get().(*bytes.Buffer)
		buffer.Reset()

		const parallel = false
		_ = encodeBranch(c, buffer, parallel)
		c.encoding = buffer.Bytes()

		var bstr string
		if len(c.encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.encoding), c.generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.encoding, c.generation)
		}

		encodingBufferPool.Put(buffer)

		sub := tree.Add(bstr)
		for i, child := range c.children {
			if child != nil {
				t.string(sub, child, i)
			}
		}
	case *leaf:
		buffer := encodingBufferPool.Get().(*bytes.Buffer)
		buffer.Reset()

		_ = encodeLeaf(c, buffer)

		c.encodingMu.Lock()
		defer c.encodingMu.Unlock()
		c.encoding = buffer.Bytes()

		var bstr string
		if len(c.encoding) > 1024 {
			bstr = fmt.Sprintf("idx=%d %s hash=%x gen=%d", idx, c.String(), common.MustBlake2bHash(c.encoding), c.generation)
		} else {
			bstr = fmt.Sprintf("idx=%d %s encode=%x gen=%d", idx, c.String(), c.encoding, c.generation)
		}

		encodingBufferPool.Put(buffer)

		tree.Add(bstr)
	default:
		return
	}
}
