// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
	Header *types.Header     `scale:"1"`
	Signer types.AuthorityID `scale:"2"`
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
	encodedHeadersWithSigners, err := s.db.Get(currentSlotKey)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return nil, fmt.Errorf("getting key slot header map key %d: %w", slot, err)
	}

	headersWithSigners := make([]headerAndSigner, 0)
	if len(encodedHeadersWithSigners) > 0 {
		encodedSliceHeadersWithSigners := make([][]byte, 0)

		err = scale.Unmarshal(encodedHeadersWithSigners, &encodedSliceHeadersWithSigners)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling encoded headers with signers: %w", err)
		}

		for _, encodedHeaderAndSigner := range encodedSliceHeadersWithSigners {
			// each header and signer instance should have an empty header
			// so we will be able to scale decode the whole byte stream with
			// the digests correctly in place
			decodedHeaderAndSigner := headerAndSigner{
				Header: types.NewEmptyHeader(),
			}

			err := scale.Unmarshal(encodedHeaderAndSigner, &decodedHeaderAndSigner)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling header with signer: %w", err)
			}

			headersWithSigners = append(headersWithSigners, decodedHeaderAndSigner)
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
	encodedHeaderAndSigner := make([][]byte, len(headersWithSigners))

	// encode each header and signer and push to a slice of bytes
	// that will be scale encoded and stored in the database
	for idx, headerAndSigner := range headersWithSigners {
		encoded, err := scale.Marshal(headerAndSigner)
		if err != nil {
			return nil, fmt.Errorf("marshalling header and signer: %w", err)
		}

		encodedHeaderAndSigner[idx] = encoded
	}

	encodedHeadersWithSigners, err = scale.Marshal(encodedHeaderAndSigner)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}

	batch := s.db.NewBatch()
	err = batch.Put(currentSlotKey, encodedHeadersWithSigners)
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

	err = batch.Flush()
	if err != nil {
		return nil, fmt.Errorf("failed to flush batch operations: %w", err)
	}

	return nil, nil
}

func saturatingSub(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return 0
}
