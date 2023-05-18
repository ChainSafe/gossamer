// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

const (
	// maxLen equivalent of `ARCH32BIT_BITSLICE_MAX_BITS` in parity-scale-codec
	maxLen = 268435455
	// byteSize is the number of bits in a byte
	byteSize = 8
)

// BitVec is the implementation of the bit vector
type BitVec struct {
	bits []bool
}

// NewBitVec returns a new BitVec with the given bits
// This isn't a complete implementation of the bit vector
// It is only used for ParachainHost runtime exports
// TODO: Implement the full bit vector
// https://github.com/ChainSafe/gossamer/issues/3248
func NewBitVec(bits []bool) BitVec {
	return BitVec{
		bits: bits,
	}
}

// Bits returns the bits in the BitVec
func (bv *BitVec) Bits() []bool {
	return bv.bits
}

// Bytes returns the byte representation of the BitVec.Bits
func (bv *BitVec) Bytes() []byte {
	return bitsToBytes(bv.bits)
}

// Size returns the number of bits in the BitVec
func (bv *BitVec) Size() uint {
	return uint(len(bv.bits))
}

// bitsToBytes converts a slice of bits to a slice of bytes
// Uses lsb ordering
// TODO: Implement msb ordering
// https://github.com/ChainSafe/gossamer/issues/3248
func bitsToBytes(bits []bool) []byte {
	bitLength := len(bits)
	numOfBytes := (bitLength + (byteSize - 1)) / byteSize
	bytes := make([]byte, numOfBytes)

	if len(bits)%byteSize != 0 {
		// Pad with zeros to make the number of bits a multiple of byteSize
		pad := make([]bool, byteSize-len(bits)%byteSize)
		bits = append(bits, pad...)
	}

	for i := 0; i < bitLength; i++ {
		if bits[i] {
			byteIndex := i / byteSize
			bitIndex := i % byteSize
			bytes[byteIndex] |= 1 << bitIndex
		}
	}

	return bytes
}

// bytesToBits converts a slice of bytes to a slice of bits
func bytesToBits(b []byte, size uint) []bool {
	var bits []bool
	for _, uint8val := range b {
		end := size
		if end > byteSize {
			end = byteSize
		}
		size -= end

		for j := uint(0); j < end; j++ {
			bit := (uint8val>>j)&1 == 1
			bits = append(bits, bit)
		}
	}

	return bits
}
