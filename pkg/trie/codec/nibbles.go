// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

// NibblesToKeyLE converts a slice of nibbles with length k into a
// Little Endian byte slice.
// It assumes nibbles are already in Little Endian and does not rearrange nibbles.
// If the length of the input is odd, the result is
// [ 0000 in[0] | in[1] in[2] | ... | in[k-2] in[k-1] ]
// Otherwise, the result is
// [ in[0] in[1] | ... | in[k-2] in[k-1] ]
func NibblesToKeyLE(nibbles []byte) []byte {
	if len(nibbles)%2 == 0 {
		keyLE := make([]byte, len(nibbles)/2)
		for i := 0; i < len(nibbles); i += 2 {
			keyLE[i/2] = (nibbles[i] << 4 & 0xf0) | (nibbles[i+1] & 0xf)
		}
		return keyLE
	}

	keyLE := make([]byte, len(nibbles)/2+1)
	keyLE[0] = nibbles[0]
	for i := 2; i < len(nibbles); i += 2 {
		keyLE[i/2] = (nibbles[i-1] << 4 & 0xf0) | (nibbles[i] & 0xf)
	}

	return keyLE
}

// KeyLEToNibbles converts a Little Endian byte slice into nibbles.
// It assumes bytes are already in Little Endian and does not rearrange nibbles.
func KeyLEToNibbles(in []byte) (nibbles []byte) {
	if len(in) == 0 {
		return []byte{}
	} else if len(in) == 1 && in[0] == 0 {
		return []byte{0, 0}
	}

	l := len(in) * 2
	nibbles = make([]byte, l)
	for i, b := range in {
		nibbles[2*i] = b / 16
		nibbles[2*i+1] = b % 16
	}

	return nibbles
}

// CommonPrefixLength returns the length of the common prefix of a and b.
func CommonPrefix(a, b []byte) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}
