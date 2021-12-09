// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrReadHeaderByte       = errors.New("cannot read header byte")
	ErrUnknownNodeType      = errors.New("unknown node type")
	ErrNodeTypeIsNotABranch = errors.New("node type is not a branch")
	ErrNodeTypeIsNotALeaf   = errors.New("node type is not a leaf")
	ErrDecodeValue          = errors.New("cannot decode value")
	ErrReadChildrenBitmap   = errors.New("cannot read children bitmap")
	ErrDecodeChildHash      = errors.New("cannot decode child hash")
)

// Decode decodes a node from a reader.
// For branch decoding, see the comments on decodeBranch.
// For leaf decoding, see the comments on decodeLeaf.
func Decode(reader io.Reader) (n Node, err error) {
	buffer := pools.SingleByteBuffers.Get().(*bytes.Buffer)
	defer pools.SingleByteBuffers.Put(buffer)
	oneByteBuf := buffer.Bytes()
	_, err = reader.Read(oneByteBuf)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadHeaderByte, err)
	}
	header := oneByteBuf[0]

	nodeType := header >> 6
	switch nodeType {
	case LeafType:
		n, err = decodeLeaf(reader, header)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case BranchType, BranchWithValueType:
		n, err = decodeBranch(reader, header)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownNodeType, nodeType)
	}
}

// decodeBranch reads and decodes from a reader with the encoding specified in lib/trie/node/encode_doc.go.
// Note that since the encoded branch stores the hash of the children nodes, we are not
// reconstructing the child nodes from the encoding. This function instead stubs where the
// children are known to be with an empty leaf. The children nodes hashes are then used to
// find other values using the persistent database.
func decodeBranch(reader io.Reader, header byte) (branch *Branch, err error) {
	nodeType := header >> 6
	if nodeType != 2 && nodeType != 3 {
		return nil, fmt.Errorf("%w: %d", ErrNodeTypeIsNotABranch, nodeType)
	}

	branch = new(Branch)

	keyLen := header & 0x3f
	branch.Key, err = decodeKey(reader, keyLen)
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

		branch.Children[i] = &Leaf{
			Hash: hash,
		}
	}

	branch.Dirty = true

	return branch, nil
}

// decodeLeaf reads and decodes from a reader with the encoding specified in lib/trie/node/encode_doc.go.
func decodeLeaf(reader io.Reader, header byte) (leaf *Leaf, err error) {
	nodeType := header >> 6
	if nodeType != 1 {
		return nil, fmt.Errorf("%w: %d", ErrNodeTypeIsNotALeaf, nodeType)
	}

	leaf = &Leaf{
		Dirty: true,
	}

	keyLen := header & 0x3f
	leaf.Key, err = decodeKey(reader, keyLen)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	sd := scale.NewDecoder(reader)
	var value []byte
	err = sd.Decode(&value)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %s", ErrDecodeValue, err)
	}

	if len(value) > 0 {
		leaf.Value = value
	}

	return leaf, nil
}
