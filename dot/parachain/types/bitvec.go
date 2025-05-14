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

// NewBitVec creates a new BitVec initialised with the given bits
func NewBitVec(bits []bool) (BitVec, error) {
	if len(bits) == 0 {
		return BitVec{}, nil
	}

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
func (bv *BitVec) Bits() []bool { // skipcq:GO-W1029
	bits := make([]bool, bv.len)
	for i := 0; i < bv.len; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		bits[i] = (bv.bits[byteIndex] & (1 << bitIndex)) != 0
	}
	return bits
}

// Len returns the number of bits in the BitVec
func (bv *BitVec) Len() int { // skipcq:GO-W1029
	return bv.len
}

// PushBits adds multiple bits to the end of the BitVec
func (bv *BitVec) PushBits(bits []bool) error { // skipcq:GO-W1029
	if len(bits) == 0 {
		return nil
	}

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

// Set sets a bit at the specified index
func (bv *BitVec) Set(index uint, bit bool) error { // skipcq:GO-W1029
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

// Get returns the bit at the specified index
func (bv *BitVec) Get(index uint) (bool, error) { // skipcq:GO-W1029
	if index >= uint(bv.len) {
		return false, errors.New("index out of bounds")
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return (bv.bits[byteIndex] & (1 << bitIndex)) != 0, nil
}

// ExtendByByte adds a byte to the BitVec. The bits are added in LSB0 order,
func (bv *BitVec) ExtendByByte(b byte) error { // skipcq:GO-W1029
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
func (bv BitVec) MarshalSCALE() ([]byte, error) { // skipcq:GO-W1029
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
func (bv *BitVec) UnmarshalSCALE(r io.Reader) error { // skipcq:GO-W1029
	if r == nil {
		return errors.New("reader is nil")
	}

	// Read the compact-encoded length
	var length uint
	err := scale.NewDecoder(r).Decode(&length)
	if err != nil {
		return fmt.Errorf("decoding compact length: %w", err)
	}

	// Check for maximum length
	if length > MaxBitVecLength {
		return fmt.Errorf("bitvec length %d exceeds maximum allowed length of %d", length, MaxBitVecLength)
	}

	// Calculate required bytes for the bits
	requiredBytes := (int(length) + 7) / 8

	// Read the required bytes for the bits
	bv.bits = make([]byte, requiredBytes)
	_, err = io.ReadFull(r, bv.bits)
	if err != nil {
		return fmt.Errorf("reading bits: %w", err)
	}

	// Update the BitVec length
	bv.len = int(length)

	return nil
}

// IsEqual checks if two BitVecs are equal by comparing their lengths and bits
func (bv *BitVec) IsEqual(other *BitVec) bool { // skipcq:GO-W1029
	if bv.len != other.len {
		return false
	}

	return bytes.Equal(bv.bits, other.bits)
}

// CountOnes returns the count of set bits (1s) in the BitVec following LSB0 ordering
func (bv *BitVec) CountOnes() int { // skipcq:GO-W1029
	if bv.len == 0 {
		return 0
	}

	var count int
	completeBytes := (bv.len + 7) / 8

	// Count ones in all bytes - no need for masking since unused bits are zero
	for i := 0; i < completeBytes; i++ {
		count += bits.OnesCount8(bv.bits[i])
	}

	return count
}

// Mask masks out bits according to the provided mask BitVec.
// For each position, the bit is kept only if it was set and the mask bit is not set.
// This implements the logic: (x, mask) => x && !mask
func (bv *BitVec) Mask(mask BitVec) { // skipcq:GO-W1029
	for i, value := range bv.Bits() {
		// (x, mask) => x
		// (true, true) => false
		// (true, false) => true
		// (false, true) => false
		// (false, false) => false

		// ignoring the error because oob in mask is equivalent to i being set to false
		m, _ := mask.Get(uint(i))
		_ = bv.Set(uint(i), value && !m) // oob is impossible
	}
}

// Clone returns a deep copy of the BitVec.
func (bv *BitVec) Clone() BitVec { // skipcq:GO-W1029
	return BitVec{
		bits: append([]byte{}, bv.bits...),
		len:  bv.len,
	}
}

// Contains checks if this BitVec contains the other BitVec as a contiguous subsequence.
// It returns true if the pattern represented by other appears anywhere within this BitVec.
// For example:
//
//	BitVec{0,0,1,0,1,1,0,0}.Contains(BitVec{0,1,1,0}) == true
//	BitVec{0,0,1,0,1,1,0,0}.Contains(BitVec{1,0,0,1}) == false
func (bv *BitVec) Contains(other BitVec) bool { // skipcq:GO-W1029
	// If other is empty, it's contained in any BitVec
	if other.Len() == 0 {
		return true
	}

	// If other is longer than bv, it can't be contained
	if other.Len() > bv.Len() {
		return false
	}

	thisBits := bv.Bits()
	otherBits := other.Bits()

	// Check each possible starting position in bv
	for i := 0; i <= bv.Len()-other.Len(); i++ {
		match := true

		// Check if other matches at this position
		for j := 0; j < other.Len(); j++ {
			if thisBits[i+j] != otherBits[j] {
				match = false
				break
			}
		}

		if match {
			return true
		}
	}

	return false
}

// Or performs a bitwise OR operation with another BitVec.
// It returns a new BitVec where a bit is set if it's set in either this BitVec or the other BitVec.
// If the BitVecs have different lengths, the result will have the length of the longer BitVec,
// and the shorter one will be treated as if padded with zeroes.
func (bv *BitVec) Or(other BitVec) BitVec { // skipcq:GO-W1029
	maxL := max(len(bv.bits), len(other.bits))
	result := make([]byte, maxL)
	copy(result, bv.bits)

	for i := 0; i < len(result); i++ {
		if i < len(other.bits) {
			result[i] |= other.bits[i]
		} else {
			break
		}
	}

	return BitVec{
		bits: result,
		len:  max(bv.len, other.len),
	}
}
