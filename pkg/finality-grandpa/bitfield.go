// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// A dynamically sized, write-once (per bit), lazily allocating bitfield.
type bitfield struct {
	bits []uint64
}

// newBitfield creates a new empty bitfield.
func newBitfield() bitfield {
	return bitfield{
		bits: make([]uint64, 0),
	}
}

// IsBlank returns Whether the bitfield is blank or empty.
func (b *bitfield) IsBlank() bool { //skipcq: GO-W1029
	return len(b.bits) == 0
}

// Merge another bitfield into this bitfield.
//
// As a result, this bitfield has all bits set that are set in either bitfield.
//
// This function only allocates if this bitfield is shorter than the other
// bitfield, in which case it is resized accordingly to accommodate for all
// bits of the other bitfield.
func (b *bitfield) Merge(other bitfield) *bitfield { //skipcq: GO-W1029
	if len(b.bits) < len(other.bits) {
		b.bits = append(b.bits, make([]uint64, len(other.bits)-len(b.bits))...)
	}
	for i, word := range other.bits {
		b.bits[i] |= word
	}
	return b
}

// SetBit will set a bit in the bitfield at the specified position.
//
// If the bitfield is not large enough to accommodate for a bit set
// at the specified position, it is resized accordingly.
func (b *bitfield) SetBit(position uint) { //skipcq: GO-W1029
	wordOff := position / 64
	bitOff := position % 64

	if wordOff >= uint(len(b.bits)) {
		newLen := wordOff + 1
		b.bits = append(b.bits, make([]uint64, newLen-uint(len(b.bits)))...)
	}
	b.bits[wordOff] |= 1 << (63 - bitOff)
}

// iter1s will get an iterator over all bits that are set (i.e. 1) in the bitfield,
// starting at bit position `start` and moving in steps of size `2^step`
// per word.
func (b *bitfield) iter1s(start, step uint) (bit1s []bit1) { //skipcq: GO-W1029
	return iter1s(b.bits, start, step)
}

// Iter1sEven will get an iterator over all bits that are set (i.e. 1) at even bit positions.
func (b *bitfield) Iter1sEven() []bit1 { //skipcq: GO-W1029
	return b.iter1s(0, 1)
}

// Iter1sOdd will get an iterator over all bits that are set (i.e. 1) at odd bit positions.
func (b *bitfield) Iter1sOdd() []bit1 { //skipcq: GO-W1029
	return b.iter1s(1, 1)
}

// iter1sMerged will get an iterator over all bits that are set (i.e. 1) when merging
// this bitfield with another bitfield, without modifying either
// bitfield, starting at bit position `start` and moving in steps
// of size `2^step` per word.
func (b *bitfield) iter1sMerged(other bitfield, start, step uint) []bit1 { //skipcq: GO-W1029
	switch {
	case len(b.bits) == len(other.bits):
		zipped := make([]uint64, len(b.bits))
		for i, a := range b.bits {
			b := other.bits[i]
			zipped[i] = a | b
		}
		return iter1s(zipped, start, step)
	case len(b.bits) < len(other.bits):
		zipped := make([]uint64, len(other.bits))
		for i, bit := range other.bits {
			var a uint64
			if i < len(b.bits) {
				a = b.bits[i]
			}
			zipped[i] = a | bit
		}
		return iter1s(zipped, start, step)
	case len(b.bits) > len(other.bits):
		zipped := make([]uint64, len(b.bits))
		for i, a := range b.bits {
			var b uint64
			if i < len(other.bits) {
				b = other.bits[i]
			}
			zipped[i] = a | b
		}
		return iter1s(zipped, start, step)
	default:
		panic("unreachable")
	}
}

// Iter1sMergedEven will get an iterator over all bits that are set (i.e. 1) at even bit positions
// when merging this bitfield with another bitfield, without modifying
// either bitfield.
func (b *bitfield) Iter1sMergedEven(other bitfield) []bit1 { //skipcq: GO-W1029
	return b.iter1sMerged(other, 0, 1)
}

// Iter1sMergedOdd will get an iterator over all bits that are set (i.e. 1) at odd bit positions
// when merging this bitfield with another bitfield, without modifying
// either bitfield.
func (b *bitfield) Iter1sMergedOdd(other bitfield) []bit1 { //skipcq: GO-W1029
	return b.iter1sMerged(other, 1, 1)
}

// Turn an iterator over u64 words into an iterator over bits that
// are set (i.e. `1`) in these words, starting at bit position `start`
// and moving in steps of size `2^step` per word.
func iter1s(iter []uint64, start, step uint) (bit1s []bit1) {
	if !(start < 64 && step < 7) {
		panic("wtf?")
	}
	steps := (64 >> step) - (start >> step)
	for i, word := range iter {
		n := steps
		if word == 0 {
			n = 0
		}
		for j := uint(0); j < n; j++ {
			bitPos := start + (j << step)
			if testBit(word, bitPos) {
				bit1s = append(bit1s, bit1{uint(i)*64 + bitPos})
			}
		}
	}
	return bit1s
}

func testBit(word uint64, position uint) bool {
	mask := uint64(1 << (63 - position))
	return word&mask == mask
}

// A bit that is set (i.e. 1) in a `bitfield`.
type bit1 struct {
	// The position of the bit in the bitfield.
	position uint
}
