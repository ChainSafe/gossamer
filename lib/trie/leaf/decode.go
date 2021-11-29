// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrReadHeaderByte     = errors.New("cannot read header byte")
	ErrNodeTypeIsNotALeaf = errors.New("node type is not a leaf")
	ErrDecodeValue        = errors.New("cannot decode value")
)

// Decode reads and decodes from a reader with the encoding specified in lib/trie/encode/doc.go.
func Decode(r io.Reader, header byte) (leaf *Leaf, err error) { // TODO return leaf
	if header == 0 { // TODO remove this is taken care of by the caller
		header, err = decode.ReadNextByte(r)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrReadHeaderByte, err)
		}
	}

	nodeType := header >> 6
	if nodeType != 1 {
		return nil, fmt.Errorf("%w: %d", ErrNodeTypeIsNotALeaf, nodeType)
	}

	leaf = new(Leaf)

	keyLen := header & 0x3f
	leaf.Key, err = decode.Key(r, keyLen)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	sd := scale.NewDecoder(r)
	var value []byte
	err = sd.Decode(&value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeValue, err)
	}

	if len(value) > 0 {
		leaf.Value = value
	}

	leaf.Dirty = true // TODO move this as soon as it gets modified

	return leaf, nil
}
