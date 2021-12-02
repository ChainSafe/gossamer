// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package encode

import (
	"errors"
	"fmt"
)

const maxPartialKeySize = ^uint16(0)

var ErrPartialKeyTooBig = errors.New("partial key length cannot be larger than or equal to 2^16")

// KeyLength encodes the public key length.
func KeyLength(keyLength int) (encoding []byte, err error) {
	keyLength -= 63

	if keyLength >= int(maxPartialKeySize) {
		return nil, fmt.Errorf("%w: %d",
			ErrPartialKeyTooBig, keyLength)
	}

	for i := uint16(0); i < maxPartialKeySize; i++ {
		if keyLength < 255 {
			encoding = append(encoding, byte(keyLength))
			break
		}
		encoding = append(encoding, byte(255))
		keyLength -= 255
	}

	return encoding, nil
}

// NibblesToKey converts a slice of nibbles with length k into a
// Big Endian byte slice.
// It assumes nibbles are already in Little Endian and does not rearrange nibbles.
// If the length of the input is odd, the result is
// [ in[1] in[0] | ... | 0000 in[k-1] ]
// Otherwise, the result is
// [ in[1] in[0] | ... | in[k-1] in[k-2] ]
func NibblesToKey(nibbles []byte) (key []byte) {
	if len(nibbles)%2 == 0 {
		key = make([]byte, len(nibbles)/2)
		for i := 0; i < len(nibbles); i += 2 {
			key[i/2] = (nibbles[i] & 0xf) | (nibbles[i+1] << 4 & 0xf0)
		}
	} else {
		key = make([]byte, len(nibbles)/2+1)
		for i := 0; i < len(nibbles); i += 2 {
			key[i/2] = nibbles[i] & 0xf
			if i < len(nibbles)-1 {
				key[i/2] |= (nibbles[i+1] << 4 & 0xf0)
			}
		}
	}

	return key
}

// NibblesToKeyLE converts a slice of nibbles with length k into a
// Little Endian byte slice.
// It assumes nibbles are already in Little Endian and does not rearrange nibbles.
// If the length of the input is odd, the result is
// [ 0000 in[0] | in[1] in[2] | ... | in[k-2] in[k-1] ]
// Otherwise, the result is
// [ in[0] in[1] | ... | in[k-2] in[k-1] ]
func NibblesToKeyLE(nibbles []byte) (keyLE []byte) {
	if len(nibbles)%2 == 0 {
		keyLE = make([]byte, len(nibbles)/2)
		for i := 0; i < len(nibbles); i += 2 {
			keyLE[i/2] = (nibbles[i] << 4 & 0xf0) | (nibbles[i+1] & 0xf)
		}
	} else {
		keyLE = make([]byte, len(nibbles)/2+1)
		keyLE[0] = nibbles[0]
		for i := 2; i < len(nibbles); i += 2 {
			keyLE[i/2] = (nibbles[i-1] << 4 & 0xf0) | (nibbles[i] & 0xf)
		}
	}

	return keyLE
}
