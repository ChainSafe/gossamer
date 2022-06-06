// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "fmt"

// Type is the type of the node.
type Type byte

const (
	// Leaf type for leaf nodes.
	Leaf Type = iota
	// Branch type for branches (with or without value).
	Branch
)

func (t Type) String() string {
	switch t {
	case Leaf:
		return "leaf"
	case Branch:
		return "branch"
	default:
		panic(fmt.Sprintf("invalid node type: %d", t))
	}
}
