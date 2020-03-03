// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package babe

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	babetypes "github.com/ChainSafe/gossamer/lib/babe/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var slotTail = uint64(12)

// returns the current slot number
func (b *Session) findCurrentSlot(slotTail uint64) (uint64, error) {
	 // find slot of chain head
	head := b.blockState.BestBlockHash()

	slot, err := b.getSlotForBlock(head)
	if err != nil {
		return 0, err
	}

	// find arrival time of chain head
	// note: this assumes that the block arrived within the slot it was produced, may be off
	arrivalTime, err := b.blockState.GetArrivalTime(hash)
	if err != nil {
		return 0, err
	}

	// use slot duration to count up
	for {
		if (arrivalTime >= time.Now().Unix() - b.config.SlotDuration) {
			return slot, nil
		}

		// increment slot, slot time
		arrivalTime += b.config.SlotDuration
		slot += 1
	}
}

// slotTime calculates the slot time in the form of miliseconds since the unix epoch
// for a given slot in miliseconds, returns 0 and an error if it can't be calculated
func (b *Session) slotTime(slot uint64, slotTail uint64) (uint64, error) {
	var at []uint64

	head := b.blockState.BestBlockHash()
	tail := new(big.Int).SetUint64(slotTail)

	deepestBlock, err := b.blockState.GetHeader(head)
	if err != nil {
		return 0, fmt.Errorf("cannot get deepest block: %s", err)
	}

	// check to make sure we have enough blocks before the deepest block to accurately calculate slot time
	if deepestBlock.Number.Cmp(tail) == -1 {
		return 0, fmt.Errorf("cannot calculate slot time: deepest block number %d less than or equal to slot tail %d", deepestBlock.Number, tail)
	}

	startNumber := tail.Sub(deepestBlock.Number, tail)

	start, err := b.blockState.GetBlockByNumber(startNumber)
	if err != nil {
		return 0, err
	}

	err = b.configurationFromRuntime()
	if err != nil {
		return 0, err
	}

	sd := b.config.SlotDuration

	var currSlot uint64
	var so uint64
	var arrivalTime uint64

	for _, hash := range b.blockState.SubChain(start.Header.Hash(), deepestBlock.Hash()) {
		currSlot, err = b.getSlotForBlock(hash)
		if err != nil {
			return 0, err
		}

		so, err = slotOffset(currSlot, slot)
		if err != nil {
			return 0, err
		}

		arrivalTime, err = b.blockState.GetArrivalTime(hash)
		if err != nil {
			return 0, err
		}

		st := arrivalTime + (so * sd)
		at = append(at, st)
	}

	st, err := median(at)
	if err != nil {
		return 0, err
	}
	return st, nil
}

// median calculates the median of a uint64 slice
// @TODO: Implement quickselect as an alternative to this.
func median(l []uint64) (uint64, error) {
	// sort the list
	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	m := len(l)
	med := uint64(0)
	if m == 0 {
		return 0, errors.New("arrival times list is empty! ")
	} else if m%2 == 0 {
		med = (l[(m/2)-1] + l[(m/2)+1]) / 2
	} else {
		med = l[m/2]
	}
	return med, nil
}

// slotOffset returns the number of slots between slot
func slotOffset(start uint64, end uint64) (uint64, error) {
	os := end - start
	if end < start {
		return 0, errors.New("cannot have negative Slot Offset! ")
	}
	return os, nil
}

// getSlotForBlock returns the slot for a block
func (b *Session) getSlotForBlock(hash common.Hash) (uint64, error) {
	header, err := b.blockState.GetHeader(hash)
	if err != nil {
		return 0, err
	}

	preDigestBytes := header.Digest[0]

	digestItem, err := types.DecodeDigestItem(preDigestBytes)
	if err != nil {
		return 0, err
	}

	preDigest, ok := digestItem.(*types.PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("first digest item is not pre-digest")
	}

	babeHeader := new(babetypes.BabeHeader)
	err = babeHeader.Decode(preDigest.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot decode babe header from pre-digest: %s", err)
	}

	return babeHeader.SlotNumber, nil
}
