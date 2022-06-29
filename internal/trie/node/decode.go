// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrDecodeValue        = errors.New("cannot decode value")
	ErrReadChildrenBitmap = errors.New("cannot read children bitmap")
	ErrDecodeChildHash    = errors.New("cannot decode child hash")
)

// Decode decodes a node from a reader.
// For branch decoding, see the comments on decodeBranch.
// For leaf decoding, see the comments on decodeLeaf.
func Decode(reader io.Reader) (n *Node, err error) {
	variant, partialKeyLength, err := decodeHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("cannot decode header: %w", err)
	}

	switch variant {
	case leafVariant.bits:
		n, err = decodeLeaf(reader, partialKeyLength)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case branchVariant.bits, branchWithValueVariant.bits:
		n, err = decodeBranch(reader, variant, partialKeyLength)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		// this is a programming error, an unknown node variant
		// should be caught by decodeHeader.
		panic(fmt.Sprintf("not implemented for node variant %08b", variant))
	}
}

// decodeBranch reads and decodes from a reader with the encoding specified in internal/trie/node/encode_doc.go.
// Note that since the encoded branch stores the hash of the children nodes, we are not
// reconstructing the child nodes from the encoding. This function instead stubs where the
// children are known to be with an empty leaf. The children nodes hashes are then used to
// find other values using the persistent database.
func decodeBranch(reader io.Reader, variant byte, partialKeyLength uint16) (
	node *Node, err error) {
	node = &Node{
		Dirty:    true,
		Children: make([]*Node, ChildrenCapacity),
	}

	node.Key, err = decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	if variant == branchWithValueVariant.bits {
		err := sd.Decode(&node.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeValue, err)
		}
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

		const hashLength = 32
		childNode := &Node{
			HashDigest: hash,
		}
		if len(hash) < hashLength {
			// Handle inlined nodes
			reader = bytes.NewReader(hash)
			variant, partialKeyLength, err := decodeHeader(reader)
			if err == nil && variant == leafVariant.bits {
				childNode, err = decodeLeaf(reader, partialKeyLength)
				if err != nil {
					return nil, fmt.Errorf("%w: at index %d: %s",
						ErrDecodeValue, i, err)
				}
			}
		}

		node.Descendants++
		node.Children[i] = childNode
	}

	return node, nil
}

// decodeLeaf reads and decodes from a reader with the encoding specified in lib/trie/node/encode_doc.go.
func decodeLeaf(reader io.Reader, partialKeyLength uint16) (node *Node, err error) {
	node = &Node{
		Dirty: true,
	}

	node.Key, err = decodeKey(reader, partialKeyLength)
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
		node.Value = value
	}

	return node, nil
}
