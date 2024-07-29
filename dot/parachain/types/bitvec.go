// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

const byteSize = 8

// BitVec is the implementation of a bit vector
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

// bitsToBytes converts a slice of bits to a slice of bytes
// Uses lsb ordering
// TODO: Implement msb ordering
// https://github.com/ChainSafe/gossamer/issues/3248
func (bv BitVec) bitsToBytes() []byte {
	bits := bv.bits
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
func (bv *BitVec) bytesToBits(b []byte) {
	var bits []bool
	var size = uint(len(b))
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
	bv.bits = bits
}

// MarshalSCALE fulfils the SCALE interface for encoding
func (bv BitVec) MarshalSCALE() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buf)
	if len(bv.bits) > 268435455 {
		return nil, fmt.Errorf("bitvec too long")
	}
	size := uint(len(bv.bits))
	err := encoder.Encode(size)
	if err != nil {
		return nil, err
	}

	bytes := bv.bitsToBytes()
	_, err = buf.Write(bytes)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalSCALE fulfils the SCALE interface for decoding
func (bv *BitVec) UnmarshalSCALE(r io.Reader) error {
	decoder := scale.NewDecoder(r)
	var bytes []byte
	err := decoder.Decode(&bytes)
	if err != nil {
		return err
	}
	bv.bytesToBits(bytes)
	return nil
}
