// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/branch"
	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
)

var (
	ErrReadHeaderByte  = errors.New("cannot read header byte")
	ErrUnknownNodeType = errors.New("unknown node type")
)

func decodeNode(reader io.Reader) (n node.Node, err error) {
	header, err := decode.ReadNextByte(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadHeaderByte, err)
	}

	nodeType := header >> 6
	switch nodeType {
	case node.LeafType:
		n, err = leaf.Decode(reader, header)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case node.BranchType, node.BranchWithValueType:
		n, err = branch.Decode(reader, header)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownNodeType, nodeType)
	}
}
