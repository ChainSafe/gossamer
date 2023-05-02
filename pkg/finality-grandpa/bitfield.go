package grandpa

// A dynamically sized, write-once (per bit), lazily allocating bitfield.
type Bitfield struct {
	bits []uint64
}

// Create a new empty bitfield.
//
// Does not allocate.
func NewBitfield() Bitfield {
	return Bitfield{
		bits: make([]uint64, 0),
	}
}

// Whether the bitfield is blank / empty.
func (b Bitfield) IsBlank() bool {
	return len(b.bits) == 0
}

// Merge another bitfield into this bitfield.
//
// As a result, this bitfield has all bits set that are set in either bitfield.
//
// This function only allocates if this bitfield is shorter than the other
// bitfield, in which case it is resized accordingly to accomodate for all
// bits of the other bitfield.
func (b *Bitfield) Merge(other Bitfield) *Bitfield {
	if len(b.bits) < len(other.bits) {
		b.bits = append(b.bits, make([]uint64, len(other.bits)-len(b.bits))...)
	}
	for i, word := range other.bits {
		b.bits[i] |= word
	}
	return b
}

// Set a bit in the bitfield at the specified position.
//
// If the bitfield is not large enough to accomodate for a bit set
// at the specified position, it is resized accordingly.
func (b *Bitfield) SetBit(position uint) {
	wordOff := position / 64
	bitOff := position % 64

	if wordOff >= uint(len(b.bits)) {
		newLen := wordOff + 1
		b.bits = append(b.bits, make([]uint64, newLen-uint(len(b.bits)))...)
	}
	b.bits[wordOff] |= 1 << (63 - bitOff)
}

// Get an iterator over all bits that are set (i.e. 1) in the bitfield,
// starting at bit position `start` and moving in steps of size `2^step`
// per word.
func (b Bitfield) iter1s(start, step uint) (bit1s []Bit1) {
	return iter1s(b.bits, start, step)
}

// Get an iterator over all bits that are set (i.e. 1) at even bit positions.
func (b Bitfield) Iter1sEven() []Bit1 {
	return b.iter1s(0, 1)
}

// Get an iterator over all bits that are set (i.e. 1) at odd bit positions.
func (b Bitfield) Iter1sOdd() []Bit1 {
	return b.iter1s(1, 1)
}

// Get an iterator over all bits that are set (i.e. 1) when merging
// this bitfield with another bitfield, without modifying either
// bitfield, starting at bit position `start` and moving in steps
// of size `2^step` per word.
func (bf Bitfield) iter1sMerged(other Bitfield, start, step uint) []Bit1 {
	switch {
	case len(bf.bits) == len(other.bits):
		zipped := make([]uint64, len(bf.bits))
		for i, a := range bf.bits {
			b := other.bits[i]
			zipped[i] = a | b
		}
		return iter1s(zipped, start, step)
	case len(bf.bits) < len(other.bits):
		zipped := make([]uint64, len(other.bits))
		for i, b := range other.bits {
			var a uint64
			if i < len(bf.bits) {
				a = bf.bits[i]
			}
			zipped[i] = a | b
		}
		return iter1s(zipped, start, step)
	case len(bf.bits) > len(other.bits):
		zipped := make([]uint64, len(bf.bits))
		for i, a := range bf.bits {
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

// Get an iterator over all bits that are set (i.e. 1) at even bit positions
// when merging this bitfield with another bitfield, without modifying
// either bitfield.
func (b Bitfield) Iter1sMergedEven(other Bitfield) []Bit1 {
	return b.iter1sMerged(other, 0, 1)
}

// Get an iterator over all bits that are set (i.e. 1) at odd bit positions
// when merging this bitfield with another bitfield, without modifying
// either bitfield.
func (b Bitfield) Iter1sMergedOdd(other Bitfield) []Bit1 {
	return b.iter1sMerged(other, 1, 1)
}

// Turn an iterator over u64 words into an iterator over bits that
// are set (i.e. `1`) in these words, starting at bit position `start`
// and moving in steps of size `2^step` per word.
func iter1s(iter []uint64, start, step uint) (bit1s []Bit1) {
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
				bit1s = append(bit1s, Bit1{uint(i)*64 + bitPos})
			}
		}
	}
	return bit1s
}

func testBit(word uint64, position uint) bool {
	mask := uint64(1 << (63 - position))
	return word&mask == mask
}

// A bit that is set (i.e. 1) in a `Bitfield`.
type Bit1 struct {
	// The position of the bit in the bitfield.
	Position uint
}
