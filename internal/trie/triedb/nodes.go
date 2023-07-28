// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"fmt"
	"strconv"

	"github.com/ChainSafe/gossamer/internal/trie/triedb/nibble"
	"github.com/qdm12/gotree"
)

type NodeType uint8

const (
	Empty NodeType = iota
	Leaf
	NibbledBranch
)

type Node struct {
	Type     NodeType
	Slice    nibble.NibbleSlice
	Value    *NodeValue
	Children []*NodeHandle
}

func NewNode(nodeType NodeType, partial nibble.NibbleSlice, value *NodeValue, children []*NodeHandle) *Node {
	return &Node{nodeType, partial, value, children}
}

func (n *Node) String() string {
	return n.StringNode().String()
}

// StringNode returns a gotree compatible node for String methods.
func (n *Node) StringNode() (stringNode *gotree.Node) {
	stringNode = gotree.New(fmt.Sprintf("%d", n.Type))
	stringNode.Appendf("Slice: %s", bytesToString(n.Slice.Data()))
	if n.Value != nil {
		stringNode.Appendf("Value: %s", bytesToString(n.Value.Data))
	} else {
		stringNode.Appendf("Value: nil")
	}
	stringNode.Appendf("Hashed: %s", strconv.FormatBool(n.Value.Hashed))
	if n.Children != nil && len(n.Children) > 0 {
		for i, child := range n.Children {
			if child == nil {
				continue
			}
			stringNode.Appendf("Child: %d", i)
			stringNode.Appendf("Child data: %s", bytesToString(child.Data))
			stringNode.Appendf("Child hashed: %s", strconv.FormatBool(child.Hashed))
		}
	}

	return stringNode
}

type NodeValue struct {
	Data   []byte
	Hashed bool
}

type NodeHandle struct {
	Data   []byte
	Hashed bool
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
