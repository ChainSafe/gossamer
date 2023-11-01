// TODO: This is just a temporary file to complete the participation module. The type definitions here are not complete.
// We need to remove this file once we have implemented the leaf update interfaces

package overseer

import "github.com/ChainSafe/gossamer/lib/common"

type LeafStatus uint

const (
	LeafStatusFresh LeafStatus = iota
	LeafStatusStale
)

type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
	Status LeafStatus
}

type ActiveLeavesUpdate struct {
	Activated *ActivatedLeaf
}
