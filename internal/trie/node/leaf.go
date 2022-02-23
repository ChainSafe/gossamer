// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/qdm12/gotree"
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
	HashDigest []byte
	Encoding   []byte
	// Generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain Generation number,
	// which is updated to match the trie Generation once they are
	// inserted, moved or iterated over.
	Generation uint64
}

// NewLeaf creates a new leaf using the arguments given.
func NewLeaf(key, value []byte, dirty bool, generation uint64) *Leaf {
	return &Leaf{
		Key:        key,
		Value:      value,
		Dirty:      dirty,
		Generation: generation,
	}
}

// Type returns LeafType.
func (l *Leaf) Type() Type {
	return LeafType
}

func (l *Leaf) String() string {
	return l.StringNode().String()
}

// StringNode returns a gotree compatible node for String methods.
func (l *Leaf) StringNode() (stringNode *gotree.Node) {
	stringNode = gotree.New("Leaf")
	stringNode.Appendf("Generation: %d", l.Generation)
	stringNode.Appendf("Dirty: %t", l.Dirty)
	stringNode.Appendf("Key: " + bytesToString(l.Key))
	stringNode.Appendf("Value: " + bytesToString(l.Value))
	stringNode.Appendf("Calculated encoding: " + bytesToString(l.Encoding))
	stringNode.Appendf("Calculated digest: " + bytesToString(l.HashDigest))
	return stringNode
}

func bytesToString(b []byte) (s string) {
	switch {
	case b == nil:
		return "nil"
	case len(b) <= 20:
		return fmt.Sprintf("0x%x", b)
	default:
		return fmt.Sprintf("0x%x...%x", b[:8], b[len(b)-8:])
	}

}
