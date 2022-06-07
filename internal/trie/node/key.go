// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
)

const maxPartialKeySize = ^uint16(0)

var (
	ErrPartialKeyTooBig = errors.New("partial key length cannot be larger than or equal to 2^16")
	ErrReadKeyLength    = errors.New("cannot read key length")
	ErrReadKeyData      = errors.New("cannot read key data")
)

// encodeKeyLength encodes the key length.
func encodeKeyLength(keyLength int, writer io.Writer) (err error) {
	keyLength -= 63

	if keyLength >= int(maxPartialKeySize) {
		return fmt.Errorf("%w: %d",
			ErrPartialKeyTooBig, keyLength)
	}

	for i := uint16(0); i < maxPartialKeySize; i++ {
		if keyLength < 255 {
			_, err = writer.Write([]byte{byte(keyLength)})
			if err != nil {
				return err
			}
			break
		}
		_, err = writer.Write([]byte{255})
		if err != nil {
			return err
		}

		keyLength -= 255
	}

	return nil
}

// decodeKey decodes a key from a reader.
func decodeKey(reader io.Reader, keyLengthByte byte) (b []byte, err error) {
	keyLength := int(keyLengthByte)

	if keyLengthByte == keyLenOffset {
		// partial key longer than 63, read next bytes for rest of pk len
		buffer := pools.SingleByteBuffers.Get().(*bytes.Buffer)
		defer pools.SingleByteBuffers.Put(buffer)
		oneByteBuf := buffer.Bytes()
		for {
			_, err = reader.Read(oneByteBuf)
			if err != nil {
				return nil, fmt.Errorf("%w: %s", ErrReadKeyLength, err)
			}
			nextKeyLen := oneByteBuf[0]

			keyLength += int(nextKeyLen)

			if nextKeyLen < 0xff {
				break
			}

			if keyLength >= int(maxPartialKeySize) {
				return nil, fmt.Errorf("%w: %d",
					ErrPartialKeyTooBig, keyLength)
			}
		}
	}

	if keyLength == 0 {
		return []byte{}, nil
	}

	key := make([]byte, keyLength/2+keyLength%2)
	n, err := reader.Read(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadKeyData, err)
	} else if n != len(key) {
		return nil, fmt.Errorf("%w: read %d bytes instead of %d",
			ErrReadKeyData, n, len(key))
	}

	return codec.KeyLEToNibbles(key)[keyLength%2:], nil
}
