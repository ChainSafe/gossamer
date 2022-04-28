// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/qdm12/gotree"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Node is a node in the trie and can be a leaf or a branch.
type Node struct {
	// Type is the node type.
	Type Type
	// Key is the partial key bytes in nibbles (0 to f in hexadecimal)
	Key   []byte
	Value []byte
	// Generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain Generation number,
	// which is updated to match the trie Generation once they are
	// inserted, moved or iterated over.
	Generation uint64
	// Children, only for branches
	Children [ChildrenCapacity]*Node

	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty bool
	// HashDigest is the cached hash digest of the
	// node encoding.
	HashDigest []byte
	// Encoding is the cached encoding of the node.
	Encoding []byte

	// Descendants is the number of descendant nodes for
	// this particular node.
	Descendants uint32
}

func (n *Node) String() string {
	return n.StringNode().String()
}

// StringNode returns a gotree compatible node for String methods.
func (n Node) StringNode() (stringNode *gotree.Node) {
	caser := cases.Title(language.BritishEnglish)
	stringNode = gotree.New(caser.String(n.Type.String()))
	stringNode.Appendf("Generation: %d", n.Generation)
	stringNode.Appendf("Dirty: %t", n.Dirty)
	stringNode.Appendf("Key: " + bytesToString(n.Key))
	stringNode.Appendf("Value: " + bytesToString(n.Value))
	stringNode.Appendf("Calculated encoding: " + bytesToString(n.Encoding))
	stringNode.Appendf("Calculated digest: " + bytesToString(n.HashDigest))

	for i, child := range n.Children {
		if child == nil {
			continue
		}
		childNode := stringNode.Appendf("Child %d", i)
		childNode.AppendNode(child.StringNode())
	}

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
