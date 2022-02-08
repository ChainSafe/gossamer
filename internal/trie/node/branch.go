// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"sync"

	"github.com/qdm12/gotree"
)

var _ Node = (*Branch)(nil)

const (
	ChildrenCapacity = 16
)

// Branch is a branch in the trie.
type Branch struct {
	// Partial key bytes in nibbles (0 to f in hexadecimal)
	Key      []byte
	Children [16]Node
	Value    []byte
	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty      bool
	HashDigest []byte
	Encoding   []byte
	// Generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain Generation number,
	// which is updated to match the trie Generation once they are
	// inserted, moved or iterated over.
	Generation uint64
	sync.RWMutex
}

// NewBranch creates a new branch using the arguments given.
func NewBranch(key, value []byte, dirty bool, generation uint64) *Branch {
	return &Branch{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		Generation: generation,
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
	return b.StringNode().String()
}

// StringNode returns a gotree compatible node for String methods.
func (b *Branch) StringNode() (stringNode *gotree.Node) {
	stringNode = gotree.New("Branch")
	stringNode.Appendf("Generation: %d", b.Generation)
	stringNode.Appendf("Dirty: %t", b.Dirty)
	stringNode.Appendf("Key: " + bytesToString(b.Key))
	stringNode.Appendf("Value: " + bytesToString(b.Value))
	stringNode.Appendf("Calculated encoding: " + bytesToString(b.Encoding))
	stringNode.Appendf("Calculated digest: " + bytesToString(b.HashDigest))

	for i, child := range b.Children {
		if child == nil {
			continue
		}
		childNode := stringNode.Appendf("Child %d", i)
		childNode.AppendNode(child.StringNode())
	}

	return stringNode
}
