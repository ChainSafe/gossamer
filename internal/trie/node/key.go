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

// SetKey sets the key to the branch.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the branch.
func (b *Branch) SetKey(key []byte) {
	b.Key = key
}

// SetKey sets the key to the leaf.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the leaf.
func (l *Leaf) SetKey(key []byte) {
	l.Key = key
}

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
func decodeKey(reader io.Reader, keyLength byte) (b []byte, err error) {
	publicKeyLength := int(keyLength)

	if keyLength == 0x3f {
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

	return codec.KeyLEToNibbles(key)[publicKeyLength%2:], nil
}
