// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package nibble

const NibblePerByte uint = 2
const PaddingBitmask byte = 0x0F
const BitPerNibble = 4
const NibbleLength = 16

// / A trie node prefix, it is the nibble path from the trie root
// / to the trie node.
// / For a node containing no partial key value it is the full key.
// / For a value node or node containing a partial key, it is the full key minus its node partial
// / nibbles (the node key can be split into prefix and node partial).
// / Therefore it is always the leftmost portion of the node key, so its internal representation
// / is a non expanded byte slice followed by a last padded byte representation.
// / The padded byte is an optional padded value.
type Prefix struct {
	PartialKey []byte
	PaddedByte *byte
}

func padLeft(b byte) byte {
	padded := (b & ^PaddingBitmask)
	return padded
}

func padRight(b byte) byte {
	padded := (b & PaddingBitmask)
	return padded
}

func NumberPadding(i uint) uint {
	return i % NibblePerByte
}

// Count the biggest common depth between two left aligned packed nibble slice
func biggestDepth(v1, v2 []byte) uint {
	upperBound := minLength(v1, v2)

	for i := uint(0); i < upperBound; i++ {
		if v1[i] != v2[i] {
			return i*NibblePerByte + leftCommon(v1[i], v2[i])
		}
	}
	return upperBound * NibblePerByte
}

// LeftCommon the number of common nibble between two left aligned bytes
func leftCommon(a, b byte) uint {
	if a == b {
		return 2
	}
	if padLeft(a) == padLeft(b) {
		return 1
	} else {
		return 0
	}
}

func minLength(v1, v2 []byte) uint {
	if len(v1) < len(v2) {
		return uint(len(v1))
	}
	return uint(len(v2))
}
