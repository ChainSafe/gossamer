// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

var _ Node = (*Branch)(nil)

// Branch is a branch in the trie.
type Branch struct {
	Key      []byte // partial key
	Children [16]Node
	Value    []byte
	// dirty is true when the branch differs
	// from the node stored in the database.
	dirty      bool
	hashDigest []byte
	Encoding   []byte
	// generation is incremented on every trie Snapshot() call.
	// Nodes that are part of the trie are then gradually updated
	// to have a matching generation number as well, if they are
	// still relevant.
	generation uint64
	sync.RWMutex
}

// NewBranch creates a new branch using the arguments given.
func NewBranch(key, value []byte, dirty bool, generation uint64) *Branch {
	return &Branch{
		Key:        key,
		Value:      value,
		dirty:      dirty,
		generation: generation,
	}
}

func (b *Branch) String() string {
	if len(b.Value) > 1024 {
		return fmt.Sprintf("key=%x childrenBitmap=%16b value (hashed)=%x dirty=%v",
			b.Key, b.ChildrenBitmap(), common.MustBlake2bHash(b.Value), b.dirty)
	}
	return fmt.Sprintf("key=%x childrenBitmap=%16b value=%v dirty=%v",
		b.Key, b.ChildrenBitmap(), b.Value, b.dirty)
}
