package state

import (
	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
)

const slotTablePrefix = "slot"

// We keep at least this number of slots in database.
const maxSlotCapacity uint64 = 1000

// We prune slots when they reach this number.
const pruningBound = 2 * maxSlotCapacity

var (
	slotHeaderMapKey = []byte("slot_header_map")
	slotHeaderStart  = []byte("slot_header_start")
)

type SlotState struct {
	db chaindb.Database
}

func NewSlotState(db *chaindb.BadgerDB) *SlotState {
	slotStateDB := chaindb.NewTable(db, slotTablePrefix)

	return &SlotState{
		db: slotStateDB,
	}
}

func (s *SlotState) checkEquivocation(header *types.Header, authorityIndex uint32,
	slot, slotNow uint64) *types.BabeEquivocationProof {

	// We don't check equivocations for old headers out of our capacity.
	// checking slotNow is greater than slot to avoid overflow, same as saturating_sub
	if slotNow > slot && (slotNow-slot) > maxSlotCapacity {
		return nil
	}

	return nil
}

func saturatingSub(a, b uint64) uint64 {

	return 0
}
