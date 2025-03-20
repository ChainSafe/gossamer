package parachaintypes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/bits"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// MaxBitVecLength is the maximum allowed length for a BitVec to prevent memory issues
const MaxBitVecLength = 1<<29 - 1 // 536870911

// BitVec represents a vector of bits with LSB0 ordering
type BitVec struct {
	bits []byte
	len  int
}

// CountOnes returns the count of set bits (1s) in the BitVec following LSB0 ordering
func (bv *BitVec) CountOnes() int {
	if bv.len == 0 {
		return 0
	}

	var count int
	completeBytes := bv.len / 8

	// Count ones in complete bytes
	for i := 0; i < completeBytes; i++ {
		count += bits.OnesCount8(bv.bits[i])
	}

	// Handle remaining bits in the last byte
	remainingBits := bv.len % 8
	if remainingBits > 0 {
		lastByte := bv.bits[completeBytes]
		// In LSB0, we want the first remainingBits from the right
		mask := byte((1 << uint(remainingBits)) - 1)
		count += bits.OnesCount8(lastByte & mask)
	}

	return count
}

// NewBitVec creates a new BitVec initialised with the given bits
func NewBitVec(bits []bool) (BitVec, error) {
	if len(bits) > MaxBitVecLength {
		return BitVec{}, fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", len(bits), MaxBitVecLength)
	}

	bv := BitVec{
		bits: make([]byte, (len(bits)+7)/8), // Allocate enough bytes to hold all bits
		len:  len(bits),
	}

	// Add each bit to the vector
	for i, bit := range bits {
		if bit {
			byteIndex := i / 8
			bitIndex := i % 8
			bv.bits[byteIndex] |= 1 << bitIndex
		}
	}

	return bv, nil
}

// Bits returns all bits in the BitVec as a slice of bools
func (bv *BitVec) Bits() []bool {
	bits := make([]bool, bv.len)
	for i := 0; i < bv.len; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bits[i] = (bv.bits[byteIndex] & (1 << bitIndex)) != 0
	}
	return bits
}

// Len returns the number of bits in the BitVec
func (bv *BitVec) Len() int {
	return bv.len
}

// PushBits adds multiple bits to the end of the BitVec
func (bv *BitVec) PushBits(bits []bool) error {
	newLength := bv.len + len(bits)

	// Check if the new length exceeds the maximum allowed length
	if newLength > MaxBitVecLength {
		return fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", newLength, MaxBitVecLength)
	}

	// Pre-allocate space if needed
	requiredBytes := (newLength + 7) / 8
	if requiredBytes > len(bv.bits) {
		bytesToAdd := make([]byte, requiredBytes-len(bv.bits))
		bv.bits = append(bv.bits, bytesToAdd...)
	}

	// Add each bit
	for _, bit := range bits {
		byteIndex := bv.len / 8
		bitIndex := bv.len % 8

		if bit {
			bv.bits[byteIndex] |= 1 << bitIndex
		}
		bv.len++
	}
	return nil
}

// SetBit sets a bit at the specified index
func (bv *BitVec) SetBit(index uint, bit bool) error {
	if index >= uint(bv.len) {
		return errors.New("index out of bounds")
	}

	byteIndex := index / 8
	bitIndex := index % 8

	if bit {
		bv.bits[byteIndex] |= 1 << bitIndex
	} else {
		bv.bits[byteIndex] &^= 1 << bitIndex
	}
	return nil
}

// GetBit returns the bit at the specified index
func (bv *BitVec) GetBit(index uint) (bool, error) {
	if index >= uint(bv.len) {
		return false, errors.New("index out of bounds")
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return (bv.bits[byteIndex] & (1 << bitIndex)) != 0, nil
}

// ExtendByByte adds a byte to the BitVec. The bits are added in LSB0 order,
func (bv *BitVec) ExtendByByte(b byte) error {
	// Check if the new length exceeds the maximum allowed length
	if bv.len+8 > MaxBitVecLength {
		return fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", bv.len+8, MaxBitVecLength)
	}

	// Pre-allocate space if needed
	requiredBytes := (bv.len + 8 + 7) / 8
	if requiredBytes > len(bv.bits) {
		bytesToAdd := make([]byte, requiredBytes-len(bv.bits))
		bv.bits = append(bv.bits, bytesToAdd...)
	}

	for i := 0; i < 8; i++ {
		byteIndex := bv.len / 8
		bitIndex := bv.len % 8

		if (b & (1 << i)) != 0 {
			bv.bits[byteIndex] |= 1 << bitIndex
		}
		bv.len++
	}
	return nil
}

// MarshalSCALE encodes the BitVec into a byte slice
func (bv BitVec) MarshalSCALE() ([]byte, error) {
	if bv.len > MaxBitVecLength {
		// as we ensure that the length is always less than MaxBitVecLength, this should never happen practically.
		// but we still check for it to prevent memory issues
		return nil, fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", bv.len, MaxBitVecLength)
	}

	length := uint(bv.len) // convert to uint for compact encoding

	header, err := scale.Marshal(length)
	if err != nil {
		return nil, fmt.Errorf("marshalling length: %w", err)
	}

	// Combine the header and the actual bits into the result
	result := make([]byte, len(header)+len(bv.bits))
	copy(result, header)                // Copy the header into the result
	copy(result[len(header):], bv.bits) // Copy the bits into the result

	return result, nil
}

// UnmarshalSCALE decodes into the BitVec
func (bv *BitVec) UnmarshalSCALE(r io.Reader) error {
	if r == nil {
		return errors.New("reader is nil")
	}

	// Read all bytes from reader
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading data: %w", err)
	}

	// Create a buffer to store the header length
	var length uint
	err = scale.Unmarshal(data, &length)
	if err != nil {
		return fmt.Errorf("unmarshalling length: %w", err)
	}

	// Check for maximum length
	if length > MaxBitVecLength {
		return fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", length, MaxBitVecLength)
	}

	// Calculate required bytes for the bits
	requiredBytes := (int(length) + 7) / 8

	// Get the header size by marshaling the length
	header, err := scale.Marshal(length)
	if err != nil {
		return fmt.Errorf("marshalling length: %w", err)
	}
	headerSize := len(header)

	// Check if we have enough data
	if len(data[headerSize:]) < requiredBytes {
		return fmt.Errorf("incomplete data: got %d bytes, expected %d", len(data[headerSize:]), requiredBytes)
	}

	// Update the BitVec with the decoded data
	bv.bits = make([]byte, requiredBytes)
	copy(bv.bits, data[headerSize:headerSize+requiredBytes])
	bv.len = int(length)

	return nil
}

// IsEqual checks if two BitVecs are equal by comparing their lengths and bits
func (bv *BitVec) IsEqual(other *BitVec) bool {
	// Check if lengths are different
	if bv.len != other.len {
		return false
	}

	// Calculate number of complete bytes to compare
	completeBytes := bv.len / 8

	// Compare complete bytes first
	if !bytes.Equal(bv.bits[:completeBytes], other.bits[:completeBytes]) {
		return false
	}

	// Check if there are any remaining bits
	remainingBits := bv.len % 8
	if remainingBits == 0 {
		return true
	}

	// Compare remaining bits in the last byte
	mask := byte((1 << remainingBits) - 1)
	lastByteIndex := completeBytes
	return (bv.bits[lastByteIndex] & mask) == (other.bits[lastByteIndex] & mask)
}
