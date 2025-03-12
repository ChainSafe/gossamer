# BitVector Implementation in Go

This package provides an efficient implementation of a bit vector (BitVec) with LSB0 (Least Significant Bit Zero) ordering. The implementation is designed to be memory-efficient and provides SCALE encoding compatibility.

## Features

- LSB0 bit ordering
- Memory-efficient storage using byte slices
- SCALE encoding/decoding support
- Dynamic resizing capabilities
- Bit-level operations
- Byte extension support

## Usage

```go
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

	// Extend the BitVec by adding bits from a byte
	bv.ExtendByByte(0b01111111) // Decimal: 254
	// bitvec will be: [true, true, true, true, false, true, false, false, true, true, true, true, true, true, true]
```

### SCALE Encoding/Decoding

The BitVec implementation supports SCALE encoding and decoding, which is compatible with Substrate's bitvec implementation:

```go
    bv := bitvec.NewBitVec([]bool{true, false, true})
    
    // Marshal to SCALE format
    encoded, err := scale.Marshal(bv)
    if err != nil {
       return fmt.Errorf("marshalling bitvec: %s", err)
    }

    // Create a new BitVec to decode into
    newBv := &bitvec.BitVec{}

	// Unmarshal from SCALE format
    err := scale.Unmarshal(encoded, &newBv)
	if err != nil {
        return fmt.Errorf("unmarshalling bitvec: %s", err)
    }
```

## Implementation Details

### Memory Layout

The BitVec stores bits in a byte slice with LSB0 ordering, meaning:
- The least significant bit (index 0) is stored in the lowest bit of the first byte
- Bits are packed into bytes to minimize memory usage
- The length is stored separately to handle non-byte-aligned bit counts

### SCALE Encoding Format

The SCALE encoding format uses a compact header to store the length, followed by the actual bits:

- For lengths < 64: Single byte header
- For lengths < 16384: Two byte header
- For lengths < 1073741824: Four byte header
- For lengths >= 1073741824: Five byte header

## Performance Considerations

- Bit operations are optimized using bitwise operations
- Memory allocation is minimized by pre-allocating space when needed
- SCALE encoding/decoding is implemented efficiently with minimal memory copies
