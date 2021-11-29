// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/node"
)

var _ node.Node = (*Branch)(nil)

// Branch is a branch in the trie.
type Branch struct {
	Key        []byte // partial key
	Children   [16]node.Node
	Value      []byte
	Dirty      bool
	Hash       []byte
	Encoding   []byte
	Generation uint64
	sync.RWMutex
}

func (b *Branch) String() string {
	if len(b.Value) > 1024 {
		return fmt.Sprintf("key=%x childrenBitmap=%16b value (hashed)=%x dirty=%v",
			b.Key, b.ChildrenBitmap(), common.MustBlake2bHash(b.Value), b.Dirty)
	}
	return fmt.Sprintf("key=%x childrenBitmap=%16b value=%v dirty=%v",
		b.Key, b.ChildrenBitmap(), b.Value, b.Dirty)
}
