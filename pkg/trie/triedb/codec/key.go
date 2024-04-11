// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/trie/codec"
)

const maxPartialKeyLength = ^uint16(0)

var ErrReaderMismatchCount = errors.New("read unexpected number of bytes from reader")

// decodeKey decodes a key from a reader.
func decodeKey(reader io.Reader, partialKeyLength uint16) (b []byte, err error) {
	if partialKeyLength == 0 {
		return []byte{}, nil
	}

	key := make([]byte, partialKeyLength/2+partialKeyLength%2)
	n, err := reader.Read(key)
	if err != nil {
		return nil, fmt.Errorf("reading from reader: %w", err)
	} else if n != len(key) {
		return nil, fmt.Errorf("%w: read %d bytes instead of expected %d bytes",
			ErrReaderMismatchCount, n, len(key))
	}

	// if the partialKeyLength is an odd number means that when parsing the key
	// to nibbles it will contains a useless 0 in the first index, otherwise
	// we can use the entire nibbles
	return codec.KeyLEToNibbles(key)[partialKeyLength%2:], nil
}
