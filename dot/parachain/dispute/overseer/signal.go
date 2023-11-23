package overseer

import "github.com/ChainSafe/gossamer/lib/common"

// Signal represents a signal sent from overseer
type Signal[data any] struct {
	Data     data
	Conclude bool
}

// LeafStatus represents the status of an activated leaf
type LeafStatus uint

const (
	// LeafStatusFresh  A leaf is fresh when it's the first time the leaf has been encountered.
	// Most leaves should be fresh.
	LeafStatusFresh LeafStatus = iota
	// LeafStatusStale A leaf is stale when it's encountered for a subsequent time. This will happen
	// when the chain is reverted or the fork-choice rule abandons some chain.
	LeafStatusStale
)

// ActivatedLeaf represents an activated leaf
type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
	Status LeafStatus
	// TODO: add more fields
}

// ActiveLeavesUpdate Changes in the set of active leaves: the parachain heads which we care to work on.
//
//	Note that the activated and deactivated fields indicate deltas, not complete sets.
//
// Subsystems should adjust their jobs to start and stop work on appropriate block hashes.
type ActiveLeavesUpdate struct {
	Activated *ActivatedLeaf
}

// BlockFinalized subsystem is informed of a finalized block by its block hash and number.
type BlockFinalized struct {
	Block
}
