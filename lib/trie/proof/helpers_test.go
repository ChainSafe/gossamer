// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/trie/node"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func padRightChildren(slice []*node.Node) (paddedSlice []*node.Node) {
	paddedSlice = make([]*node.Node, node.ChildrenCapacity)
	copy(paddedSlice, slice)
	return paddedSlice
}

func encodeNode(t *testing.T, node node.Node) (encoded []byte) {
	t.Helper()
	buffer := bytes.NewBuffer(nil)
	err := node.Encode(buffer)
	require.NoError(t, err)
	return buffer.Bytes()
}

func blake2bNode(t *testing.T, node node.Node) (digest []byte) {
	t.Helper()
	encoding := encodeNode(t, node)
	digestHash, err := common.Blake2bHash(encoding)
	require.NoError(t, err)
	digest = digestHash[:]
	return digest
}
