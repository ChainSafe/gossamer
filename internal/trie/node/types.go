// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

// Type is the byte type for the node.
type Type byte

const (
	_ Type = iota
	// LeafType type is 1
	LeafType
	// BranchType type is 2
	BranchType
	// BranchWithValueType type is 3
	BranchWithValueType
)
