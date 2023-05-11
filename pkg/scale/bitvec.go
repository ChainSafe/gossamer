// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

const (
	// maxLen equivalent of `ARCH32BIT_BITSLICE_MAX_BITS` in parity-scale-codec
	maxLen = 268435455
	// byteSize is the number of bits in a byte
	byteSize = 8
)

// BitVec represents rust's `bitvec::BitVec` in SCALE
// It is encoded as a compact u32 representing the number of bits in the vector
// followed by the actual bits, rounded up to the nearest byte
type BitVec interface {
	// Bits returns the bits in the BitVec
	Bits() []uint8
	// Bytes returns the byte representation of the Bits
	Bytes() []byte
	// Size returns the number of bits in the BitVec
	Size() uint
}

// bitVec implements BitVec
type bitVec struct {
	size uint    `scale:"1"`
	bits []uint8 `scale:"2"`
}

// NewBitVec returns a new BitVec with the given bits
func NewBitVec(bits []uint8) BitVec {
	var size uint
	if bits != nil {
		size = uint(len(bits))
	}

	return &bitVec{
		size: size,
		bits: bits,
	}
}

// Bits returns the bits in the BitVec
func (bv *bitVec) Bits() []uint8 {
	return bv.bits
}

// Bytes returns the byte representation of the BitVec.Bits
func (bv *bitVec) Bytes() []byte {
	return bitsToBytes(bv.bits)
}

// Size returns the number of bits in the BitVec
func (bv *bitVec) Size() uint {
	return bv.size
}

// bitsToBytes converts a slice of bits to a slice of bytes
func bitsToBytes(bits []uint8) []byte {
	bitLength := len(bits)
	numOfBytes := (bitLength + (byteSize - 1)) / byteSize
	bytes := make([]byte, numOfBytes)

	if len(bits)%byteSize != 0 {
		// Pad with zeros to make the number of bits a multiple of byteSize
		pad := make([]uint8, byteSize-len(bits)%byteSize)
		bits = append(bits, pad...)
	}

	for i := 0; i < bitLength; i++ {
		if bits[i] == 1 {
			byteIndex := i / byteSize
			bitIndex := i % byteSize
			bytes[byteIndex] |= 1 << bitIndex
		}
	}

	return bytes
}

// bytesToBits converts a slice of bytes to a slice of bits
func bytesToBits(b []byte, size uint) []uint8 {
	var bits []uint8
	for _, uint8val := range b {
		end := size
		if end > byteSize {
			end = byteSize
		}
		size -= end

		for j := uint(0); j < end; j++ {
			bit := (uint8val >> j) & 1
			bits = append(bits, bit)
		}
	}

	return bits
}
