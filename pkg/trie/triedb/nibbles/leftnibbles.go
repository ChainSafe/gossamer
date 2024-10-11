// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibbles

import (
	"bytes"
	"cmp"
)

// A representation of a nibble slice which is left-aligned. The regular [Nibbles] is
// right-aligned, meaning it does not support efficient truncation from the right side.
//
// This is meant to be an immutable struct. No operations actually change it.
type LeftNibbles struct {
	bytes []byte
	len   uint
}

// Constructs a byte-aligned nibble slice from a byte slice.
func NewLeftNibbles(bytes []byte) LeftNibbles {
	return LeftNibbles{
		bytes: bytes,
		len:   uint(len(bytes)) * NibblesPerByte,
	}
}

// Returns the length of the slice in nibbles.
func (ln LeftNibbles) Len() uint {
	return ln.len
}

func leftNibbleAt(v1 []byte, ix uint) uint8 {
	return atLeft(uint8(ix%NibblesPerByte), v1[ix/NibblesPerByte]) //nolint:gosec
}

// Get the nibble at a nibble index padding with a 0 nibble. Returns nil if the index is
// out of bounds.
func (ln LeftNibbles) At(index uint) *uint8 {
	if index < ln.len {
		at := leftNibbleAt(ln.bytes, index)
		return &at
	}
	return nil
}

// Returns a new slice truncated from the right side to the given length. If the given length
// is greater than that of this slice, the function just returns a copy.
func (ln LeftNibbles) Truncate(len uint) LeftNibbles {
	if ln.len < len {
		len = ln.len
	}
	return LeftNibbles{bytes: ln.bytes, len: len}
}

// Returns whether the given slice is a prefix of this one.
func (ln LeftNibbles) StartsWith(prefix LeftNibbles) bool {
	return ln.Truncate(prefix.Len()).compare(prefix) == 0
}

// Returns whether another regular (right-aligned) nibble slice is contained in this one at
// the given offset.
func (ln LeftNibbles) Contains(partial Nibbles, offset uint) bool {
	for i := uint(0); i < partial.Len(); i++ {
		lnAt := ln.At(offset + i)
		partialAt := partial.At(i)
		if *lnAt == partialAt {
			continue
		}
		return false
	}
	return true
}

func (ln LeftNibbles) compare(other LeftNibbles) int {
	commonLen := ln.Len()
	if other.Len() < commonLen {
		commonLen = other.Len()
	}
	commonByteLen := commonLen / NibblesPerByte

	// Quickly compare the common prefix of the byte slices.
	c := bytes.Compare(ln.bytes[:commonByteLen], other.bytes[:commonByteLen])
	if c != 0 {
		return c
	}

	// Compare nibble-by-nibble (either 0 or 1 nibbles) any after the common byte prefix.
	for i := commonByteLen * NibblesPerByte; i < commonLen; i++ {
		a := *ln.At(i)
		b := *other.At(i)
		if c := cmp.Compare(a, b); c != 0 {
			return c
		}
	}

	// If common nibble prefix is the same, finally compare lengths.
	return cmp.Compare(ln.Len(), other.Len())
}
