package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const slotTablePrefix = "slot"

// We keep at least this number of slots in database.
const maxSlotCapacity uint64 = 1000

// We prune slots when they reach this number.
const pruningBound = 2 * maxSlotCapacity

var (
	slotHeaderMapKey   = []byte("slot_header_map")
	slotHeaderStartKey = []byte("slot_header_start")
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

type headerAndSigner struct {
	Header *types.Header
	Signer types.AuthorityID
}

func (s *SlotState) CheckEquivocation(slotNow, slot uint64, header *types.Header,
	signer types.AuthorityID) (*types.BabeEquivocationProof, error) {
	// We don't check equivocations for old headers out of our capacity.
	// checking slotNow is greater than slot to avoid overflow, same as saturating_sub
	if saturatingSub(slotNow, slot) > maxSlotCapacity {
		return nil, nil
	}

	slotEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotEncoded, slot)

	currentSlotKey := bytes.Join([][]byte{slotHeaderMapKey, slotEncoded[:]}, nil)
	encodedheadersWithSigners, err := s.db.Get(currentSlotKey)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return nil, fmt.Errorf("getting key slot header map key %d: %w", slot, err)
	}

	headersWithSigners := make([]headerAndSigner, 0)
	if len(encodedheadersWithSigners) > 0 {
		err = scale.Unmarshal(encodedheadersWithSigners, &headersWithSigners)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling headers with signers: %w", err)
		}
	}

	firstSavedSlot := slot
	firstSavedSlotEncoded, err := s.db.Get(slotHeaderStartKey)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return nil, fmt.Errorf("getting key slot header start key: %w", err)
	}

	if len(firstSavedSlotEncoded) > 0 {
		firstSavedSlot = binary.LittleEndian.Uint64(firstSavedSlotEncoded)
	}

	if slotNow < firstSavedSlot {
		// The code below assumes that slots will be visited sequentially.
		return nil, nil
	}

	for _, headerAndSigner := range headersWithSigners {
		// A proof of equivocation consists of two headers:
		// 1) signed by the same voter,
		if headerAndSigner.Signer == signer {
			// 2) with different hash
			if headerAndSigner.Header.Hash() != header.Hash() {
				return &types.BabeEquivocationProof{
					Slot:         slot,
					Offender:     signer,
					FirstHeader:  *headerAndSigner.Header,
					SecondHeader: *header,
				}, nil
			} else {
				// We don't need to continue in case of duplicated header,
				// since it's already saved and a possible equivocation
				// would have been detected before.
				return nil, nil
			}
		}
	}

	keysToDelete := make([][]byte, 0)
	newFirstSavedSlot := firstSavedSlot

	if slotNow-firstSavedSlot >= pruningBound {
		newFirstSavedSlot = saturatingSub(slotNow, maxSlotCapacity)

		for s := firstSavedSlot; s < newFirstSavedSlot; s++ {
			slotEncoded := make([]byte, 8)
			binary.LittleEndian.PutUint64(slotEncoded, s)

			toDelete := bytes.Join([][]byte{slotHeaderMapKey, slotEncoded[:]}, nil)
			keysToDelete = append(keysToDelete, toDelete)
		}
	}

	headersWithSigners = append(headersWithSigners, headerAndSigner{Header: header, Signer: signer})
	encodedheadersWithSigners, err = scale.Marshal(headersWithSigners)
	if err != nil {
		return nil, fmt.Errorf("marshaling: %w", err)
	}

	batch := s.db.NewBatch()
	err = batch.Put(currentSlotKey, encodedheadersWithSigners)
	if err != nil {
		return nil, fmt.Errorf("while batch putting encoded headers with signers: %w", err)
	}

	newFirstSavedSlotEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(newFirstSavedSlotEncoded, newFirstSavedSlot)
	err = batch.Put(slotHeaderStartKey, newFirstSavedSlotEncoded)
	if err != nil {
		return nil, fmt.Errorf("while batch putting encoded new first saved slot: %w", err)
	}

	for _, toDelete := range keysToDelete {
		err := batch.Del(toDelete)
		if err != nil {
			return nil, fmt.Errorf("while batch deleting key %s: %w", string(toDelete), err)
		}
	}

	batch.Flush()
	return nil, nil
}

func saturatingSub(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}
