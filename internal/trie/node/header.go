// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"errors"
	"fmt"
	"io"
)

// encodeHeader writes the encoded header for the node.
func encodeHeader(node *Node, writer io.Writer) (err error) {
	partialKeyLength := len(node.PartialKey)
	if partialKeyLength > int(maxPartialKeyLength) {
		panic(fmt.Sprintf("partial key length is too big: %d", partialKeyLength))
	}

	// Merge variant byte and partial key length together
	var variant variant
	if node.Kind() == Leaf {
		variant = leafVariant
	} else if node.SubValue == nil {
		variant = branchVariant
	} else {
		variant = branchWithValueVariant
	}

	buffer := make([]byte, 1)
	buffer[0] = variant.bits
	partialKeyLengthMask := ^variant.mask

	if partialKeyLength < int(partialKeyLengthMask) {
		// Partial key length fits in header byte
		buffer[0] |= byte(partialKeyLength)
		_, err = writer.Write(buffer)
		return err
	}

	// Partial key length does not fit in header byte only
	buffer[0] |= partialKeyLengthMask
	partialKeyLength -= int(partialKeyLengthMask)
	_, err = writer.Write(buffer)
	if err != nil {
		return err
	}

	for {
		buffer[0] = 255
		if partialKeyLength < 255 {
			buffer[0] = byte(partialKeyLength)
		}

		_, err = writer.Write(buffer)
		if err != nil {
			return err
		}

		partialKeyLength -= int(buffer[0])

		if buffer[0] < 255 {
			break
		}
	}

	return nil
}

var (
	ErrPartialKeyTooBig = errors.New("partial key length cannot be larger than 2^16")
)

func decodeHeader(reader io.Reader) (variant byte,
	partialKeyLength uint16, err error) {
	buffer := make([]byte, 1)
	_, err = reader.Read(buffer)
	if err != nil {
		return 0, 0, fmt.Errorf("reading header byte: %w", err)
	}

	variant, partialKeyLengthHeader, partialKeyLengthHeaderMask,
		err := decodeHeaderByte(buffer[0])
	if err != nil {
		return 0, 0, fmt.Errorf("decoding header byte: %w", err)
	}

	partialKeyLength = uint16(partialKeyLengthHeader)
	if partialKeyLengthHeader < partialKeyLengthHeaderMask {
		// partial key length is contained in the first byte.
		return variant, partialKeyLength, nil
	}

	// the partial key length header byte is equal to its maximum
	// possible value; this means the partial key length is greater
	// than this (0 to 2^6 - 1 = 63) maximum value, and we need to
	// accumulate the next bytes from the reader to get the full
	// partial key length.
	// Specification: https://spec.polkadot.network/#defn-node-header
	var previousKeyLength uint16 // used to track an eventual overflow
	for {
		_, err = reader.Read(buffer)
		if err != nil {
			return 0, 0, fmt.Errorf("reading key length: %w", err)
		}

		previousKeyLength = partialKeyLength
		partialKeyLength += uint16(buffer[0])

		if partialKeyLength < previousKeyLength {
			// the partial key can have a length up to 65535 which is the
			// maximum uint16 value; therefore if we overflowed, we went over
			// this maximum.
			overflowed := maxPartialKeyLength - previousKeyLength + partialKeyLength
			return 0, 0, fmt.Errorf("%w: overflowed by %d", ErrPartialKeyTooBig, overflowed)
		}

		if buffer[0] < 255 {
			// the end of the partial key length has been reached.
			return variant, partialKeyLength, nil
		}
	}
}

var ErrVariantUnknown = errors.New("node variant is unknown")

// variantsOrderedByBitMask is an array of all variants sorted
// in ascending order by the number of LHS set bits each variant mask has.
// See https://spec.polkadot.network/#defn-node-header
// WARNING: DO NOT MUTATE.
// This array is defined at global scope for performance
// reasons only, instead of having it locally defined in
// the decodeHeaderByte function below.
// For 7 variants, the performance is improved by ~20%.
var variantsOrderedByBitMask = [...]variant{
	leafVariant,            // mask 1100_0000
	branchVariant,          // mask 1100_0000
	branchWithValueVariant, // mask 1100_0000
}

func decodeHeaderByte(header byte) (variantBits,
	partialKeyLengthHeader, partialKeyLengthHeaderMask byte, err error) {
	for i := len(variantsOrderedByBitMask) - 1; i >= 0; i-- {
		variantBits = header & variantsOrderedByBitMask[i].mask
		if variantBits != variantsOrderedByBitMask[i].bits {
			continue
		}

		partialKeyLengthHeaderMask = ^variantsOrderedByBitMask[i].mask
		partialKeyLengthHeader = header & partialKeyLengthHeaderMask
		return variantBits, partialKeyLengthHeader,
			partialKeyLengthHeaderMask, nil
	}

	return 0, 0, 0, fmt.Errorf("%w: for header byte %08b", ErrVariantUnknown, header)
}
