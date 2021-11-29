// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package decode

import (
	"errors"
	"fmt"
	"io"
)

const maxPartialKeySize = ^uint16(0)

var (
	ErrPartialKeyTooBig = errors.New("partial key length cannot be larger than or equal to 2^16")
	ErrReadKeyLength    = errors.New("cannot read key length")
	ErrReadKeyData      = errors.New("cannot read key data")
)

// Key decodes a key from a reader.
func Key(reader io.Reader, keyLength byte) (b []byte, err error) {
	publicKeyLength := int(keyLength)

	if keyLength == 0x3f {
		// partial key longer than 63, read next bytes for rest of pk len
		for {
			nextKeyLen, err := ReadNextByte(reader)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrReadKeyLength, err)
			}
			publicKeyLength += int(nextKeyLen)

			if nextKeyLen < 0xff {
				break
			}

			if publicKeyLength >= int(maxPartialKeySize) {
				return nil, fmt.Errorf("%w: %d",
					ErrPartialKeyTooBig, publicKeyLength)
			}
		}
	}

	if publicKeyLength == 0 {
		return []byte{}, nil
	}

	key := make([]byte, publicKeyLength/2+publicKeyLength%2)
	n, err := reader.Read(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadKeyData, err)
	} else if n != len(key) {
		return nil, fmt.Errorf("%w: read %d bytes instead of %d",
			ErrReadKeyData, n, len(key))
	}

	return KeyLEToNibbles(key)[publicKeyLength%2:], nil
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
