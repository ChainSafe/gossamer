// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/branch"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie/pools"
)

var (
	ErrReadHeaderByte  = errors.New("cannot read header byte")
	ErrUnknownNodeType = errors.New("unknown node type")
)

func decodeNode(reader io.Reader) (n node.Node, err error) {
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
