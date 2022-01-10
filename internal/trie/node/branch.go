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
	// Partial key bytes in nibbles (0 to f in hexadecimal)
	Key      []byte
	Children [16]Node
	Value    []byte
	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty      bool
	hashDigest []byte
	Encoding   []byte
	// generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain generation number,
	// which is updated to match the trie generation once they are
	// inserted, moved or iterated over.
	generation uint64
	sync.RWMutex
}

// NewBranch creates a new branch using the arguments given.
func NewBranch(key, value []byte, dirty bool, generation uint64) *Branch {
	return &Branch{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		generation: generation,
	}
}

// Type returns BranchType if the branch value
// is nil, and BranchWithValueType otherwise.
func (b *Branch) Type() Type {
	if b.Value == nil {
		return BranchType
	}
	return BranchWithValueType
}

func (b *Branch) String() string {
	if len(b.Value) > 1024 {
		return fmt.Sprintf("branch key=0x%x childrenBitmap=%b value (hashed)=0x%x dirty=%t",
			b.Key, b.ChildrenBitmap(), common.MustBlake2bHash(b.Value), b.Dirty)
	}
	return fmt.Sprintf("branch key=0x%x childrenBitmap=%b value=0x%x dirty=%t",
		b.Key, b.ChildrenBitmap(), b.Value, b.Dirty)
}
