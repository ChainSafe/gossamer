// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/hash"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
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

// Decode decodes a node from a reader.
// The encoding format is documented in the README.md
// of this package, and specified in the Polkadot spec at
// https://spec.polkadot.network/chap-state#defn-node-header
func Decode[H hash.Hash](reader io.Reader) (n EncodedNode, err error) {
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
		n, err = decodeLeaf[H](reader, variant, partialKey)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case branchVariant, branchWithValueVariant, branchWithHashedValueVariant:
		n, err = decodeBranch[H](reader, variant, partialKey)
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
func decodeBranch[H hash.Hash](reader io.Reader, variant variant, partialKey nibbles.Nibbles) (
	node Branch, err error) {
	node = Branch{
		PartialKey: partialKey,
	}

	var childrenBitmap uint16
	err = binary.Read(reader, binary.LittleEndian, &childrenBitmap)
	if err != nil {
		return Branch{}, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	switch variant {
	case branchWithValueVariant:
		valueBytes := make([]byte, 0)
		err := sd.Decode(&valueBytes)
		if err != nil {
			return Branch{}, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}

		node.Value = InlineValue(valueBytes)
	case branchWithHashedValueVariant:
		hashedValue, err := decodeHashedValue[H](reader)
		if err != nil {
			return Branch{}, err
		}
		node.Value = HashedValue[H]{hashedValue}
	default:
		// Do nothing, branch without value
	}

	for i := 0; i < ChildrenCapacity; i++ {
		// Skip this index if we don't have a child here
		if (childrenBitmap>>i)&1 != 1 {
			continue
		}

		var hash []byte
		err := sd.Decode(&hash)
		if err != nil {
			return Branch{}, fmt.Errorf("%w: at index %d: %s",
				ErrDecodeChildHash, i, err)
		}

		if len(hash) < (*new(H)).Length() {
			node.Children[i] = InlineNode(hash)
		} else {
			var h H
			err := scale.Unmarshal(hash, &h)
			if err != nil {
				panic(err)
			}
			node.Children[i] = HashedNode[H]{h}
		}
	}

	return node, nil
}

// decodeLeaf reads from a reader and decodes to a leaf node.
func decodeLeaf[H hash.Hash](reader io.Reader, variant variant, partialKey nibbles.Nibbles) (node Leaf, err error) {
	node = Leaf{
		PartialKey: partialKey,
	}

	sd := scale.NewDecoder(reader)

	if variant == leafWithHashedValueVariant {
		hashedValue, err := decodeHashedValue[H](sd)
		if err != nil {
			return Leaf{}, err
		}

		node.Value = HashedValue[H]{hashedValue}
		return node, nil
	}

	valueBytes := make([]byte, 0)
	err = sd.Decode(&valueBytes)
	if err != nil {
		return Leaf{}, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}

	node.Value = InlineValue(valueBytes)

	return node, nil
}

func decodeHashedValue[H hash.Hash](reader io.Reader) (hash H, err error) {
	buffer := make([]byte, (*new(H)).Length())
	n, err := reader.Read(buffer)
	if err != nil {
		return hash, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}
	if n < (*new(H)).Length() {
		return hash, fmt.Errorf("%w: expected %d, got: %d", ErrDecodeHashedValueTooShort, (*new(H)).Length(), n)
	}

	// return buffer, nil
	h := new(H)
	err = scale.Unmarshal(buffer, h)
	return *h, err
}
