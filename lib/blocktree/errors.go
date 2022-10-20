// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blocktree

import "fmt"

// ErrParentNotFound is returned if the parent hash does not exist in the blocktree
var (
	ErrParentNotFound = fmt.Errorf("cannot find parent block in blocktree")

	// ErrBlockExists is returned if attempting to re-add a block
	ErrBlockExists = fmt.Errorf("cannot add block to blocktree that already exists")

	// ErrStartNodeNotFound is returned if the start of a subchain does not exist
	ErrStartNodeNotFound = fmt.Errorf("start node does not exist")

	// ErrEndNodeNotFound is returned if the end of a subchain does not exist
	ErrEndNodeNotFound = fmt.Errorf("end node does not exist")

	// ErrNilDescendant is returned if calling subchain with a nil node
	ErrNilDescendant = fmt.Errorf("descendant node is nil")

	// ErrDescendantNotFound is returned if a descendant in a subchain cannot be found
	ErrDescendantNotFound = fmt.Errorf("could not find descendant node")

	// ErrNodeNotFound is returned if a node with given hash doesn't exist
	ErrNodeNotFound = fmt.Errorf("could not find node")

	// ErrFailedToGetRuntime is returned when runtime doesn't exist in blockTree for corresponding block.
	ErrFailedToGetRuntime = fmt.Errorf("failed to get runtime instance")

	// ErrNumGreaterThanHighest is returned when attempting to get a
	// hash by number that is higher than any in the blocktree
	ErrNumGreaterThanHighest = fmt.Errorf("cannot find node with number greater than highest in blocktree")

	// ErrNumLowerThanRoot is returned when attempting to get a hash by number that is lower than the root node
	ErrNumLowerThanRoot = fmt.Errorf("cannot find node with number lower than root node")

	// ErrNoCommonAncestor is returned when a common ancestor cannot be found between two nodes
	ErrNoCommonAncestor = fmt.Errorf("no common ancestor between two nodes")

	errUnexpectedNumber = fmt.Errorf("block number is not parent number + 1")
)
