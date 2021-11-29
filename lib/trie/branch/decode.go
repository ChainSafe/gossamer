// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrReadHeaderByte       = errors.New("cannot read header byte")
	ErrNodeTypeIsNotABranch = errors.New("node type is not a branch")
	ErrReadChildrenBitmap   = errors.New("cannot read children bitmap")
	ErrDecodeValue          = errors.New("cannot decode value")
	ErrDecodeChildHash      = errors.New("cannot decode child hash")
)

// Decode reads and decodes from a reader with the encoding specified in lib/trie/encode/doc.go.
// Note that since the encoded branch stores the hash of the children nodes, we are not
// reconstructing the child nodes from the encoding. This function instead stubs where the
// children are known to be with an empty leaf. The children nodes hashes are then used to
// find other values using the persistent database.
func Decode(reader io.Reader, header byte) (branch *Branch, err error) {
	if header == 0 { // TODO remove this is taken care of by the caller
		header, err = decode.ReadNextByte(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrReadHeaderByte, err)
		}
	}

	nodeType := header >> 6
	if nodeType != 2 && nodeType != 3 {
		return nil, fmt.Errorf("%w: %d", ErrNodeTypeIsNotABranch, nodeType)
	}

	branch = new(Branch)

	keyLen := header & 0x3f
	branch.Key, err = decode.Key(reader, keyLen)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	if nodeType == 3 {
		var value []byte
		// branch w/ value
		err := sd.Decode(&value)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeValue, err)
		}
		branch.Value = value
	}

	for i := 0; i < 16; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 != 1 {
			continue
		}
		var hash []byte
		err := sd.Decode(&hash)
		if err != nil {
			return nil, fmt.Errorf("%w: at index %d: %s",
				ErrDecodeChildHash, i, err)
		}

		branch.Children[i] = &leaf.Leaf{
			Hash: hash,
		}
	}

	branch.Dirty = true // TODO move as soon as it gets modified?

	return branch, nil
}
