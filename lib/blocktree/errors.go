package blocktree

import (
	"errors"
)

// ErrParentNotFound is returned if the parent hash does not exist in the blocktree
var (
	ErrParentNotFound = errors.New("cannot find parent block in blocktree")

	// ErrBlockExists is returned if attempting to re-add a block
	ErrBlockExists = errors.New("cannot add block to blocktree that already exists")

	// ErrStartNodeNotFound is returned if the start of a subchain does not exist
	ErrStartNodeNotFound = errors.New("start node does not exist")

	// ErrEndNodeNotFound is returned if the end of a subchain does not exist
	ErrEndNodeNotFound = errors.New("end node does not exist")

	// ErrNilDatabase is returned in the database is nil
	ErrNilDatabase = errors.New("blocktree database is nil")

	// ErrNilDescendant is returned if calling subchain with a nil node
	ErrNilDescendant = errors.New("descendant node is nil")

	// ErrDescendantNotFound is returned if a descendant in a subchain cannot be found
	ErrDescendantNotFound = errors.New("could not find descendant node")

	// ErrNodeNotFound is returned if a node with given hash doesn't exist
	ErrNodeNotFound = errors.New("could not find node")

	// ErrFailedToGetRuntime is returned when runtime doesn't exist in blockTree for corresponding block.
	ErrFailedToGetRuntime = errors.New("failed to get runtime instance")

	// ErrNumGreaterThanHighest is returned when attempting to get a hash by number that is higher than any in the blocktree
	ErrNumGreaterThanHighest = errors.New("cannot find node with number greater than highest in blocktree")

	// ErrNumLowerThanRoot is returned when attempting to get a hash by number that is lower than the root node
	ErrNumLowerThanRoot = errors.New("cannot find node with number lower than root node")

	// ErrNoCommonAncestor is returned when a common ancestor cannot be found between two nodes
	ErrNoCommonAncestor = errors.New("no common ancestor between two nodes")

	errUnexpectedNumber = errors.New("block number is not parent number + 1")
)
