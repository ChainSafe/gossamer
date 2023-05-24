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
	// ErrDecodeStorageValue is defined since no sentinel error is defined
	// in the scale package.
	ErrDecodeStorageValue = errors.New("cannot decode storage value")
	ErrReadChildrenBitmap = errors.New("cannot read children bitmap")
	// ErrDecodeChildHash is defined since no sentinel error is defined
	// in the scale package.
	ErrDecodeChildHash = errors.New("cannot decode child hash")
)

// Decode decodes a node from a reader.
// The encoding format is documented in the README.md
// of this package, and specified in the Polkadot spec at
// https://spec.polkadot.network/#sect-state-storage
// For branch decoding, see the comments on decodeBranch.
// For leaf decoding, see the comments on decodeLeaf.
func Decode(reader io.Reader) (n *Node, err error) {
	variant, partialKeyLength, err := decodeHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}

	switch variant {
	case emptyVariant:
		return nil, nil //nolint:nilnil
	case leafVariant:
		n, err = decodeLeaf(reader, partialKeyLength)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case branchVariant, branchWithValueVariant:
		n, err = decodeBranch(reader, variant, partialKeyLength)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		// this is a programming error, an unknown node variant should be caught by decodeHeader.
		panic(fmt.Sprintf("not implemented for node variant %08b", variant))
	}
}

// decodeBranch reads from a reader and decodes to a node branch.
// Note that since the encoded branch stores the hash of the children nodes, we are not
// reconstructing the child nodes from the encoding. This function instead stubs where the
// children are known to be with an empty leaf. The children nodes hashes are then used to
// find other storage values using the persistent database.
func decodeBranch(reader io.Reader, variant variant, partialKeyLength uint16) (
	node *Node, err error) {
	node = &Node{
		Children: make([]*Node, ChildrenCapacity),
	}

	node.PartialKey, err = decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	if variant == branchWithValueVariant {
		err := sd.Decode(&node.StorageValue)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}
	}

	for i := 0; i < ChildrenCapacity; i++ {
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
			MerkleValue: hash,
		}
		if len(hash) < hashLength {
			// Handle inlined nodes
			reader = bytes.NewReader(hash)
			childNode, err = Decode(reader)
			if err != nil {
				return nil, fmt.Errorf("decoding inlined child at index %d: %w", i, err)
			}
			node.Descendants += childNode.Descendants
		}

		node.Descendants++
		node.Children[i] = childNode
	}

	return node, nil
}

// decodeLeaf reads from a reader and decodes to a leaf node.
func decodeLeaf(reader io.Reader, partialKeyLength uint16) (node *Node, err error) {
	node = &Node{}

	node.PartialKey, err = decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	sd := scale.NewDecoder(reader)
	err = sd.Decode(&node.StorageValue)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}

	return node, nil
}
