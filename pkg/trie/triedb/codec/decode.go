// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	ErrDecodeHashedStorageValue  = errors.New("cannot decode hashed storage value")
	ErrDecodeHashedValueTooShort = errors.New("hashed storage value too short")
	ErrReadChildrenBitmap        = errors.New("cannot read children bitmap")
	// ErrDecodeChildHash is defined since no sentinel error is defined
	// in the scale package.
	ErrDecodeChildHash = errors.New("cannot decode child hash")
	// ErrDecodeStorageValue is defined since no sentinel error is defined
	// in the scale package.
	ErrDecodeStorageValue = errors.New("cannot decode storage value")
)

const hashLength = common.HashLength

// Decode decodes a node from a reader.
// The encoding format is documented in the README.md
// of this package, and specified in the Polkadot spec at
// https://spec.polkadot.network/chap-state#defn-node-header
func Decode(reader io.Reader) (n Node, err error) {
	variant, partialKeyLength, err := decodeHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}

	if variant == emptyVariant {
		return Empty{}, nil
	}

	partialKey, err := decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	switch variant {
	case leafVariant, leafWithHashedValueVariant:
		n, err = decodeLeaf(reader, variant, partialKey)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case branchVariant, branchWithValueVariant, branchWithHashedValueVariant:
		n, err = decodeBranch(reader, variant, partialKey)
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
// Note that we are not decoding the children nodes.
func decodeBranch(reader io.Reader, variant variant, partialKey []byte) (
	node Branch, err error) {
	node = Branch{
		PartialKey: partialKey,
	}

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return Branch{}, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	switch variant {
	case branchWithValueVariant:
		valueBytes := make([]byte, 0)
		err := sd.Decode(&valueBytes)

		node.Value = NewInlineValue(valueBytes)
		if err != nil {
			return Branch{}, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}
	case branchWithHashedValueVariant:
		hashedValue, err := decodeHashedValue(reader)
		if err != nil {
			return Branch{}, err
		}
		node.Value = NewHashedValue(hashedValue)
	default:
		// Do nothing, branch without value
	}

	for i := 0; i < ChildrenCapacity; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 != 1 {
			continue
		}

		var hash []byte
		err := sd.Decode(&hash)
		if err != nil {
			return Branch{}, fmt.Errorf("%w: at index %d: %s",
				ErrDecodeChildHash, i, err)
		}

		if len(hash) < hashLength {
			node.Children[i] = NewInlineNode(hash)
		} else {
			node.Children[i] = NewHashedNode(hash)
		}
	}

	return node, nil
}

// decodeLeaf reads from a reader and decodes to a leaf node.
func decodeLeaf(reader io.Reader, variant variant, partialKey []byte) (node Leaf, err error) {
	node = Leaf{
		PartialKey: partialKey,
	}

	sd := scale.NewDecoder(reader)

	if variant == leafWithHashedValueVariant {
		hashedValue, err := decodeHashedValue(reader)
		if err != nil {
			return Leaf{}, err
		}
		node.Value = NewHashedValue(hashedValue)
		return node, nil
	}

	valueBytes := make([]byte, 0)
	err = sd.Decode(&valueBytes)
	if err != nil {
		return Leaf{}, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}

	node.Value = NewInlineValue(valueBytes)

	return node, nil
}

func decodeHashedValue(reader io.Reader) ([]byte, error) {
	buffer := make([]byte, hashLength)
	n, err := reader.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}
	if n < hashLength {
		return nil, fmt.Errorf("%w: expected %d, got: %d", ErrDecodeHashedValueTooShort, hashLength, n)
	}

	return buffer, nil
}
