package parachaintypes

import (
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common/math"
)

// BitVec represents a vector of bits with LSB0 ordering
type BitVec struct {
	bits []byte
	len  int
}

// NewBitVec creates a new BitVec initialized with the given bits
func NewBitVec(bits []bool) BitVec {
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

	return bv
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
func (bv *BitVec) PushBits(bits []bool) {
	// Pre-allocate space if needed
	requiredBytes := (bv.len + len(bits) + 7) / 8
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
}

// SetBit sets a bit at the specified index
func (bv *BitVec) SetBit(index int, bit bool) error {
	if index >= bv.len {
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
func (bv *BitVec) GetBit(index int) (bool, error) {
	if index >= bv.len {
		return false, errors.New("index out of bounds")
	}

	byteIndex := index / 8
	bitIndex := index % 8

	return (bv.bits[byteIndex] & (1 << bitIndex)) != 0, nil
}

// ExtendByByte adds a byte to the BitVec. The bits are added in LSB0 order,
func (bv *BitVec) ExtendByByte(b byte) {
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
}

// MarshalSCALE encodes the BitVec into a byte slice
func (bv BitVec) MarshalSCALE() ([]byte, error) {
	length := uint32(bv.len) // Get the current length of bits
	var header []byte        // This will hold the compact length encoding

	// Encoding logic based on the length of the BitVec
	if length < 64 {
		// For lengths less than 64, encode as: (length << 2)
		header = []byte{byte(length << 2)}
	} else if length < 16384 {
		// For lengths between 64 and 16384, encode as: ((length - 64) << 2) | 0b01
		adjusted := length - 64
		header = []byte{
			byte((adjusted << 2) | 0b01), // Set the lowest bit
			byte(adjusted >> 6),          // Next byte for overflow
		}
	} else if length < 1073741824 {
		// For lengths between 16384 and 1073741824, encode as: ((length - 16384) << 2) | 0b10
		adjusted := length - 16384
		header = []byte{
			byte((adjusted << 2) | 0b10), // Set the second lowest bit
			byte(adjusted >> 6),
			byte(adjusted >> 14),
			byte(adjusted >> 22),
		}
	} else {
		// For lengths 1073741824 and above, encode as: ((length - 1073741824) << 2) | 0b11
		adjusted := length - 1073741824
		header = []byte{
			byte((adjusted << 2) | 0b11), // Set both lowest bits
			byte(adjusted >> 6),
			byte(adjusted >> 14),
			byte(adjusted >> 22),
			byte(adjusted >> 30),
		}
	}

	// Combine the header and the actual bits into the result
	result := make([]byte, len(header)+len(bv.bits))
	copy(result, header)                // Copy the header into the result
	copy(result[len(header):], bv.bits) // Copy the bits into the result

	return result, nil
}

// UnmarshalSCALE decodes a SCALE encoded byte slice into a BitVec
func (bv *BitVec) UnmarshalSCALE(r io.Reader) error {
	// Read first byte for mode and initial length bits
	firstByte := make([]byte, 1)
	if _, err := r.Read(firstByte); err != nil {
		return fmt.Errorf("failed to read first byte: %w", err)
	}

	// Get the mode bits (lowest 2 bits) and the length bits
	mode := firstByte[0] & 0b11
	lengthBits := firstByte[0] >> 2

	var length uint32

	// Decode length based on mode
	switch mode {
	case 0:
		// Simple mode: length is in the top 6 bits
		length = uint32(lengthBits)
	case 1:
		// Two byte mode: length is in top 6 bits + next byte
		nextByte := make([]byte, 1)
		if _, err := r.Read(nextByte); err != nil {
			return fmt.Errorf("failed to read second byte: %w", err)
		}
		length = uint32(lengthBits) | (uint32(nextByte[0]) << 6)
		length += 64 // Add offset for two byte mode
	case 2:
		// Four byte mode: length is in top 6 bits + next 3 bytes
		nextBytes := make([]byte, 3)
		if _, err := r.Read(nextBytes); err != nil {
			return fmt.Errorf("failed to read next 3 bytes: %w", err)
		}
		length = uint32(lengthBits) |
			(uint32(nextBytes[0]) << 6) |
			(uint32(nextBytes[1]) << 14) |
			(uint32(nextBytes[2]) << 22)
		length += 16384 // Add offset for four byte mode
	case 3:
		// Five byte mode: length is in top 6 bits + next 4 bytes
		nextBytes := make([]byte, 4)
		if _, err := r.Read(nextBytes); err != nil {
			return fmt.Errorf("failed to read next 4 bytes: %w", err)
		}
		length = uint32(lengthBits) |
			(uint32(nextBytes[0]) << 6) |
			(uint32(nextBytes[1]) << 14) |
			(uint32(nextBytes[2]) << 22) |
			(uint32(nextBytes[3]) << 30)
		length += 1073741824 // Add offset for five byte mode
	default:
		return errors.New("invalid mode bits")
	}

	// Calculate required bytes for the bits and read them
	requiredBytes := (int(length) + 7) / 8
	bits := make([]byte, requiredBytes)
	if _, err := r.Read(bits); err != nil {
		return fmt.Errorf("failed to read bits: %w", err)
	}

	// Update the BitVec with the decoded data
	bv.bits = bits
	bv.len = int(length)

	return nil
}

func Yup() {
	// Create a new BitVec with initial bits
	bits := []bool{true, false, true, true, false}
	bv := NewBitVec(bits)

	// Get length
	fmt.Printf("Length: %d\n", bv.Len()) // Output: Length: 5

	// Get bit at index
	if bit, err := bv.GetBit(2); err == nil {
		fmt.Printf("Bit at index 2: %v\n", bit) // Output: Bit at index 2: true
	}

	// Set bit at index
	bv.SetBit(1, true) // bitvec will be: [true, true, true, true, false]

	// Add more bits
	bv.PushBits([]bool{true, false}) // bitvec will be: [true, true, true, true, false, true, false]

	// Get all bits
	allBits := bv.Bits() // Output: [true, true, true, true, false, true, false]
	fmt.Printf("All bits: %v\n", allBits)

	// // Extract last 8 bits from a decimal number
	// number := uint32(305419896)           // Binary: 0b00011111010110100011010010001000
	// lastBits, err := GetLast8Bits(number) // Gets binary: 0b01111000 (decimal: 120)
	// if err != nil {
	// 	panic(err)
	// }

	number1 := uint32(255)

	// number1Byte := byte(number1)
	if math.MaxUint8 >= number1 {
		bv.ExtendByByte(byte(number1))
	} else {
		println("number1 is too large")
	}

	number2 := uint32(256)
	if math.MaxUint8 >= number2 {
		bv.ExtendByByte(byte(number2))
	} else {
		println("number2 is too large")
	}

	// Add the extracted bits to BitVec
	// bv.ExtendByByte(lastBits)
}
