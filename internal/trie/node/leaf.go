// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
)

var _ Node = (*Leaf)(nil)

// Leaf is a leaf in the trie.
type Leaf struct {
	// Partial key bytes in nibbles (0 to f in hexadecimal)
	Key   []byte
	Value []byte
	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty      bool
	hashDigest []byte
	Encoding   []byte
	encodingMu sync.RWMutex
	// generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain generation number,
	// which is updated to match the trie generation once they are
	// inserted, moved or iterated over.
	generation uint64
	sync.RWMutex
}

// NewLeaf creates a new leaf using the arguments given.
func NewLeaf(key, value []byte, dirty bool, generation uint64) *Leaf {
	return &Leaf{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		generation: generation,
	}
}

// Type returns LeafType.
func (l *Leaf) Type() Type {
	return LeafType
}

func (l *Leaf) String() string {
	if len(l.Value) > 1024 {
		return fmt.Sprintf("leaf key=0x%x value (hashed)=0x%x dirty=%t", l.Key, common.MustBlake2bHash(l.Value), l.Dirty)
	}
	return fmt.Sprintf("leaf key=0x%x value=0x%x dirty=%t", l.Key, l.Value, l.Dirty)
}
