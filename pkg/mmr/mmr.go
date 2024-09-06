// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package mmr

import (
	"errors"
	"math/bits"
)

var (
	errorInconsistentStore = errors.New("inconsistent store")
	errorGetRootOnEmpty    = errors.New("get root on empty MMR")
	errorNotEnoughPeeks    = errors.New("not enough peaks")
)

type MMRStorage[T any] interface {
	getElement(pos uint64) (*T, error)
	append(pos uint64, elements []T) error
	commit() error
}

type MergeFunc[T any] func(left, right T) (*T, error)

// MMR represents a Merkle Mountain Range (MMR) which is a persistent,
// append-only data structure that allows for efficient cryptographic proofs of
// inclusion for any piece of data added to it.
type MMR[T any] struct {
	size    uint64
	storage MMRStorage[T]
	merge   MergeFunc[T]
}

// NewMMR initialises and returns a new MMR instance.
func NewMMR[T any](size uint64, storage MMRStorage[T], merger MergeFunc[T]) *MMR[T] {
	return &MMR[T]{
		size:    size,
		storage: storage,
		merge:   merger,
	}
}

// Push adds a new leaf to the MMR returning its position.
func (mmr *MMR[T]) Push(leaf T) (uint64, error) {
	elements := []T{leaf}
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

		rightElement := elements[len(elements)-1]

		parentElement, err := mmr.merge(*leftElement, rightElement)
		if err != nil {
			return 0, err
		}

		if err != nil {
			return 0, err
		}

		elements = append(elements, *parentElement)
	}

	err := mmr.storage.append(elemPosition, elements)
	if err != nil {
		return 0, err
	}

	mmr.size = position + 1
	return position, nil
}

// Root returns the root of the MMR by merging the peaks.
func (mmr *MMR[T]) Root() (*T, error) {
	if mmr.size == 0 {
		return nil, errorGetRootOnEmpty
	} else if mmr.size == 1 {
		root, err := mmr.storage.getElement(0)
		if err != nil || root == nil {
			return nil, errorInconsistentStore
		}
		return root, nil
	}

	peaksPosition := mmr.getPeaks()
	peaks := make([]T, 0)

	for _, pos := range peaksPosition {
		peak, err := mmr.storage.getElement(pos)
		if err != nil || peak == nil {
			return nil, errorInconsistentStore
		}
		peaks = append(peaks, *peak)
	}

	return mmr.bagPeaks(peaks)
}

// Commit commits the current state of the MMR to underlying storage.
func (mmr *MMR[T]) Commit() error {
	return mmr.storage.commit()
}

func (mmr *MMR[T]) findElement(position uint64, values []T) (*T, error) {
	if position > mmr.size {
		positionOffset := position - mmr.size
		return &values[positionOffset], nil
	}

	value, err := mmr.storage.getElement(position)
	if err != nil || value == nil {
		return nil, errorInconsistentStore
	}

	return value, nil
}

/*
Returns a bitmap of the peaks in the MMR.
Eg: 0b11 means that the MMR has 2 peaks at position 0 and at position 1
*/
func (mmr *MMR[T]) peakMap() uint64 {
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
func (mmr *MMR[T]) getPeaks() []uint64 {
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

func (mmr *MMR[T]) bagPeaks(peaks []T) (*T, error) {
	for len(peaks) > 1 {
		var rightPeak, leftPeak T

		rightPeak, peaks = peaks[len(peaks)-1], peaks[:len(peaks)-1]
		leftPeak, peaks = peaks[len(peaks)-1], peaks[:len(peaks)-1]

		mergedPeak, err := mmr.merge(rightPeak, leftPeak)
		if err != nil {
			return nil, err
		}
		peaks = append(peaks, *mergedPeak)
	}

	if len(peaks) < 1 {
		return nil, errorNotEnoughPeeks
	}

	// #nosec G602
	return &peaks[0], nil
}
