// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrPartialKeyTooBig = errors.New("partial key length cannot be larger than 2^16")
	ErrVariantUnknown   = errors.New("node variant is unknown")
)

func decodeHeader(reader io.Reader) (nodeVariant variant,
	partialKeyLength uint16, err error) {
	buffer := make([]byte, 1)
	_, err = reader.Read(buffer)
	if err != nil {
		return nodeVariant, 0, fmt.Errorf("reading header byte: %w", err)
	}

	nodeVariant, partialKeyLengthHeader, err := decodeHeaderByte(buffer[0])
	if err != nil {
		return variant{}, 0, fmt.Errorf("decoding header byte: %w", err)
	}

	partialKeyLengthHeaderMask := nodeVariant.partialKeyLengthHeaderMask()
	if partialKeyLengthHeaderMask == emptyVariant.bits {
		// empty node or compact encoding which have no
		// partial key. The partial key length mask is
		// 0b0000_0000 since the variant mask is
		// 0b1111_1111.
		return nodeVariant, 0, nil
	}

	partialKeyLength = uint16(partialKeyLengthHeader)
	if partialKeyLengthHeader < partialKeyLengthHeaderMask {
		// partial key length is contained in the first byte.
		return nodeVariant, partialKeyLength, nil
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
			return variant{}, 0, fmt.Errorf("reading key length: %w", err)
		}

		previousKeyLength = partialKeyLength
		partialKeyLength += uint16(buffer[0])

		if partialKeyLength < previousKeyLength {
			// the partial key can have a length up to 65535 which is the
			// maximum uint16 value; therefore if we overflowed, we went over
			// this maximum.
			overflowed := maxPartialKeyLength - previousKeyLength + partialKeyLength
			return variant{}, 0, fmt.Errorf("%w: overflowed by %d", ErrPartialKeyTooBig, overflowed)
		}

		if buffer[0] < 255 {
			// the end of the partial key length has been reached.
			return nodeVariant, partialKeyLength, nil
		}
	}
}

// variantsOrderedByBitMask is an array of all variants sorted
// in ascending order by the number of LHS set bits each variant mask has.
// See https://spec.polkadot.network/#defn-node-header
// WARNING: DO NOT MUTATE.
// This array is defined at global scope for performance
// reasons only, instead of having it locally defined in
// the decodeHeaderByte function below.
// For 7 variants, the performance is improved by ~20%.
var variantsOrderedByBitMask = [...]variant{
	leafVariant,                  // mask 1100_0000
	branchVariant,                // mask 1100_0000
	branchWithValueVariant,       // mask 1100_0000
	leafWithHashedValueVariant,   // mask 1110_0000
	branchWithHashedValueVariant, // mask 1111_0000
	emptyVariant,                 // mask 1111_1111
	compactEncodingVariant,       // mask 1111_1111
}

func decodeHeaderByte(header byte) (nodeVariant variant,
	partialKeyLengthHeader byte, err error) {
	var partialKeyLengthHeaderMask byte
	for i := len(variantsOrderedByBitMask) - 1; i >= 0; i-- {
		nodeVariant = variantsOrderedByBitMask[i]
		variantBits := header & nodeVariant.mask
		if variantBits != nodeVariant.bits {
			continue
		}

		partialKeyLengthHeaderMask = nodeVariant.partialKeyLengthHeaderMask()
		partialKeyLengthHeader = header & partialKeyLengthHeaderMask
		return nodeVariant, partialKeyLengthHeader, nil
	}

	return invalidVariant, 0, fmt.Errorf("%w: for header byte %08b", ErrVariantUnknown, header)
}
