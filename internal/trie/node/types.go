// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "fmt"

// Kind is the type of the node.
type Kind byte

const (
	// Leaf kind for leaf nodes.
	Leaf Kind = iota
	// Branch kind for branches (with or without value).
	Branch
)

func (k Kind) String() string {
	switch k {
	case Leaf:
		return "Leaf"
	case Branch:
		return "Branch"
	default:
		panic(fmt.Sprintf("invalid node type: %d", k))
	}
}
