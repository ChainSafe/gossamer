// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Package node defines the `Node` structure with methods
// to be used in the modified Merkle-Patricia Radix-16 trie.
package node

import (
	"fmt"

	"github.com/qdm12/gotree"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Node is a node in the trie and can be a leaf or a branch.
type Node struct {
	// PartialKey is the partial key bytes in nibbles (0 to f in hexadecimal)
	PartialKey          []byte
	StorageValue        []byte
	StorageValueInlined bool
	// Generation is incremented on every trie Snapshot() call.
	// Each node also contain a certain Generation number,
	// which is updated to match the trie Generation once they are
	// inserted, moved or iterated over.
	Generation uint64
	// Children is a slice of length 16 for branches.
	// It is left to nil for leaves to reduce memory usage.
	Children []*Node

	// Dirty is true when the branch differs
	// from the node stored in the database.
	Dirty bool
	// MerkleValue is the cached Merkle value of the node.
	MerkleValue []byte

	// Descendants is the number of descendant nodes for
	// this particular node.
	Descendants uint32
}

// Kind returns Leaf or Branch depending on what kind
// the node is.
func (n *Node) Kind() Kind {
	if n.Children != nil {
		return Branch
	}
	return Leaf
}

func (n *Node) String(database Getter) string {
	return n.StringNode(database).String()
}

// StringNode returns a gotree compatible node for String methods.
func (n Node) StringNode(database Getter) (stringNode *gotree.Node) {
	caser := cases.Title(language.BritishEnglish)
	stringNode = gotree.New(caser.String(n.Kind().String()))
	stringNode.Appendf("Generation: %d", n.Generation)
	stringNode.Appendf("Dirty: %t", n.Dirty)
	stringNode.Appendf("Partial key: " + bytesToString(n.PartialKey))

	subValue, err := n.GetStorageValue(database)
	if err != nil {
		panic(err)
	}
	subValueOrigin := "loaded from database"
	if n.StorageValueInlined {
		subValueOrigin = "inlined"
	}
	stringNode.Appendf("Storage value: " + bytesToString(subValue) + " (" + subValueOrigin + ")")

	if n.Descendants > 0 { // must be a branch
		stringNode.Appendf("Descendants: %d", n.Descendants)
	}
	stringNode.Appendf("Merkle value: " + bytesToString(n.MerkleValue))

	for i, child := range n.Children {
		if child == nil {
			continue
		}
		childNode := stringNode.Appendf("Child %d", i)
		childNode.AppendNode(child.StringNode(database))
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
