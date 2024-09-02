// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

/*
	Package inspired in:
	https://github.com/paritytech/merkle-mountain-range/blob/8a8a2dd5d172545faac314f3e7b6a43a85395c03/src/mmr.rs
	https://github.com/mimblewimble/grin/blob/845c41de13e9bdeb0f0b4667fbc7ef8be921a2f4/core/src/core/pmmr/pmmr.rs
*/

package mmr

import (
	"errors"
	"hash"
	"math/bits"
	"sync"
)

var (
	errorInconsistentStore = errors.New("inconsistent store")
	errorGetRootOnEmpty    = errors.New("get root on empty MMR")
)

type MMRElement []byte

type MMRNode struct {
	pos      uint64
	elements []MMRElement
}

type MMR struct {
	size   uint64
	batch  *MMRBatch
	hasher hash.Hash
	mtx    sync.Mutex
}

func NewMMR(size uint64, batch *MMRBatch, hasher hash.Hash) *MMR {
	return &MMR{
		size:   size,
		batch:  batch,
		hasher: hasher,
	}
}

// Push adds a new leaf to the MMR returning its position.
func (mmr *MMR) Push(leaf MMRElement) (uint64, error) {
	elements := []MMRElement{leaf}
	peakMap := mmr.peakMap()
	elemPosition := mmr.size
	position := mmr.size
	peak := uint64(1)

	for (peakMap & peak) != 0 {
		peak <<= 1
		position += 1
		leftPosition := position - peak
		leftElement, err := mmr.findElement(leftPosition, elements)

		if err != nil {
			return 0, err
		}

		rightElement := elements[len(elements)-1] // TODO: check this wont fail

		parentElement := mmr.merge(leftElement, rightElement)

		if err != nil {
			return 0, err
		}

		elements = append(elements, parentElement)
	}

	mmr.batch.append(elemPosition, elements)
	mmr.size = position + 1
	return position, nil
}

// Root returns the root of the MMR.
// This is doing by bagging the peaks and merging them.
func (mmr *MMR) Root() (MMRElement, error) {
	if mmr.size == 0 {
		return nil, errorGetRootOnEmpty
	} else if mmr.size == 1 {
		root, err := mmr.batch.getElement(0)
		if err != nil || root == nil {
			return nil, errorInconsistentStore
		}
		return *root, nil
	}

	peaksPosition := mmr.getPeaks()
	peaks := make([]MMRElement, 0)

	for _, pos := range peaksPosition {
		peak, err := mmr.batch.getElement(pos)
		if err != nil || peak == nil {
			return nil, errorInconsistentStore
		}
		peaks = append(peaks, *peak)
	}

	return mmr.bagPeaks(peaks), nil
}

func (mmr *MMR) findElement(position uint64, values []MMRElement) (MMRElement, error) {
	if position > mmr.size {
		positionOffset := position - mmr.size
		return values[positionOffset], nil
	}

	value, err := mmr.batch.getElement(position)
	if err != nil || value == nil {
		return nil, errorInconsistentStore
	}

	return *value, nil
}

func (mmr *MMR) merge(left, right MMRElement) MMRElement {
	// Since we could share mmr.hash instance in multiple goroutines
	defer mmr.mtx.Unlock()
	mmr.mtx.Lock()

	mmr.hasher.Reset()
	mmr.hasher.Write(left)
	mmr.hasher.Write(right)
	return mmr.hasher.Sum(nil)
}

/*
Returns a bitmap of the peaks in the MMR.
Eg: 0b11 means that the MMR has 2 peaks at position 0 and at position 1
*/
func (mmr *MMR) peakMap() uint64 {
	if mmr.size == 0 {
		return 0
	}

	pos := mmr.size
	peakSize := ^uint64(0) >> bits.LeadingZeros64(pos)
	peakMap := uint64(0)

	for peakSize > 0 {
		peakMap <<= 1
		if pos >= peakSize {
			pos -= peakSize
			peakMap |= 1
		}
		peakSize >>= 1
	}

	return peakMap
}

/*
getPeaks() the positions of the peaks in the MMR.
*/
func (mmr *MMR) getPeaks() []uint64 {
	if mmr.size == 0 {
		return []uint64{}
	}

	pos := mmr.size
	peakSize := ^uint64(0) >> bits.LeadingZeros64(pos)
	peaks := make([]uint64, 0)
	peaksSum := uint64(0)
	for peakSize > 0 {
		if pos >= peakSize {
			pos -= peakSize
			peaks = append(peaks, peaksSum+peakSize-1)
			peaksSum += peakSize
		}
		peakSize >>= 1
	}

	return peaks
}

func (mmr *MMR) bagPeaks(peaks []MMRElement) MMRElement {
	for len(peaks) > 1 {
		var rightPeak, leftPeak MMRElement

		rightPeak, peaks = peaks[len(peaks)-1], peaks[:len(peaks)-1]
		leftPeak, peaks = peaks[len(peaks)-1], peaks[:len(peaks)-1]

		mergedPeak := mmr.merge(leftPeak, rightPeak)
		peaks = append(peaks, mergedPeak)
	}

	if len(peaks) < 1 {
		return nil
	}

	// #nosec G602
	return peaks[0]
}
