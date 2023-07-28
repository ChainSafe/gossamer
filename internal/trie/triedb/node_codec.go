// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/internal/trie/triedb/nibble"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var EmptyNode = &Node{}

var (
	ErrDecodeStorageValue        = errors.New("cannot decode storage value")
	ErrDecodeHashedValueTooShort = errors.New("hashed storage value too short")
	ErrReadChildrenBitmap        = errors.New("cannot read children bitmap")
	ErrDecodeChildHash           = errors.New("cannot decode child hash")
)

func Decode(reader io.Reader) (n *Node, err error) {
	variant, nibbleCount, err := node.DecodeHeader(reader)

	logger.Errorf("Nibble count %d", nibbleCount)

	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}
	switch variant {
	case node.EmptyVariant:
		return EmptyNode, nil
	case node.LeafVariant, node.LeafWithHashedValueVariant:
		n, err = decodeLeaf(reader, variant, nibbleCount)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case node.BranchVariant, node.BranchWithValueVariant, node.BranchWithHashedValueVariant:
		n, err = decodeBranch(reader, variant, nibbleCount)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		// this is a programming error, an unknown node variant should be caught by decodeHeader.
		panic(fmt.Sprintf("not implemented for node variant %08b", variant))
	}
}

func decodeBranch(reader io.Reader, variant node.Variant, nibbleCount uint16) (*Node, error) {
	//padding := nibbleCount%uint16(nibble.NibblePerByte) != 0

	/*buffer := make([]byte, 1)
	_, err := reader.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("reading header byte: %w", err)
	}*/

	/*if padding && nibble.PadLeft(buffer[0]) != 0 {
		return nil, fmt.Errorf("bad format")
	}*/

	partial, err := node.DecodeKey(reader, nibbleCount)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	partialPadding := nibble.NumberPadding(uint(nibbleCount))

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)
	nodeValue := &NodeValue{}

	switch variant {
	case node.BranchWithValueVariant:
		err := sd.Decode(nodeValue.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}
	case node.BranchWithHashedValueVariant:
		nodeValue, err = decodeHashedValue(reader)
		if err != nil {
			return nil, err
		}
	case node.BranchVariant:
		nodeValue = nil
	default:
		// Ignored
	}

	children := make([]*NodeHandle, node.ChildrenCapacity)

	for i := 0; i < node.ChildrenCapacity; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 != 1 {
			continue
		}

		var hash []byte
		err := sd.Decode(&hash)
		if err != nil {
			return nil, fmt.Errorf("%w: at index %d: %s",
				ErrDecodeChildHash, i, err)
		}

		children[i] = &NodeHandle{
			Data:   hash,
			Hashed: (len(hash) == common.HashLength),
		}
	}

	return NewNode(NibbledBranch, *nibble.NewNibbleSliceWithPadding(partial, partialPadding), nodeValue, children), nil
}

func decodeLeaf(reader io.Reader, variant node.Variant, nibbleCount uint16) (*Node, error) {
	padding := nibbleCount%uint16(nibble.NibblePerByte) != 0

	buffer := make([]byte, 1)
	_, err := reader.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("reading header byte: %w", err)
	}

	if padding && nibble.PadLeft(buffer[0]) != 0 {
		return nil, fmt.Errorf("bad format")
	}

	partial, err := node.DecodeKey(reader, nibbleCount)

	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	partialPadding := nibble.NumberPadding(uint(nibbleCount))

	nodeValue := &NodeValue{}

	if variant == node.LeafVariant {
		sd := scale.NewDecoder(reader)
		sd.Decode(nodeValue.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}
		return NewNode(Leaf, *nibble.NewNibbleSliceWithPadding(partial, partialPadding), nodeValue, nil), nil
	}

	nodeValue, err = decodeHashedValue(reader)

	if err != nil {
		return nil, err
	}

	return NewNode(Leaf, *nibble.NewNibbleSliceWithPadding(partial, partialPadding), nodeValue, nil), nil
}

func decodeHashedValue(reader io.Reader) (*NodeValue, error) {
	buffer := make([]byte, common.HashLength)
	n, err := reader.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}
	if n < common.HashLength {
		return nil, fmt.Errorf("%w: expected %d, got: %d", ErrDecodeHashedValueTooShort, common.HashLength, n)
	}

	return &NodeValue{buffer, true}, nil
}
