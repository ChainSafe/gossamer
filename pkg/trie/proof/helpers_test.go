// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/node"
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
	err := node.Encode(buffer, trie.NoMaxInlineValueSize)
	require.NoError(t, err)
	return buffer.Bytes()
}

func blake2bNode(t *testing.T, node node.Node) (digest []byte) {
	t.Helper()
	encoding := encodeNode(t, node)
	return blake2b(t, encoding)
}

func scaleEncode(t *testing.T, data []byte) (encoded []byte) {
	t.Helper()
	encoded, err := scale.Marshal(data)
	require.NoError(t, err)
	return encoded
}

func blake2b(t *testing.T, data []byte) (digest []byte) {
	t.Helper()
	digestHash, err := common.Blake2bHash(data)
	require.NoError(t, err)
	digest = digestHash[:]
	return digest
}

func concatBytes(slices [][]byte) (concatenated []byte) {
	for _, slice := range slices {
		concatenated = append(concatenated, slice...)
	}
	return concatenated
}

// generateBytes generates a pseudo random byte slice
// of the given length. It uses `0` as its seed so
// calling it multiple times will generate the same
// byte slice. This is designed as such in order to have
// deterministic unit tests.
func generateBytes(t *testing.T, length uint) (bytes []byte) {
	t.Helper()
	generator := rand.New(rand.NewSource(0)) //skipcq: GSC-G404
	bytes = make([]byte, length)
	_, err := generator.Read(bytes)
	require.NoError(t, err)
	return bytes
}

// getBadNodeEncoding returns a particular bad node encoding of 33 bytes.
func getBadNodeEncoding() (badEncoding []byte) {
	return []byte{
		0x3, 0x94, 0xfd, 0xc2, 0xfa, 0x2f, 0xfc, 0xc0, 0x41, 0xd3,
		0xff, 0x12, 0x4, 0x5b, 0x73, 0xc8, 0x6e, 0x4f, 0xf9, 0x5f,
		0xf6, 0x62, 0xa5, 0xee, 0xe8, 0x2a, 0xbd, 0xf4, 0x4a, 0x2d,
		0xb, 0x75, 0xfb}
}

func Test_getBadNodeEncoding(t *testing.T) {
	t.Parallel()

	badEncoding := getBadNodeEncoding()
	_, err := node.Decode(bytes.NewBuffer(badEncoding))
	require.Error(t, err)
}

func assertLongEncoding(t *testing.T, node node.Node) {
	t.Helper()

	encoding := encodeNode(t, node)
	require.Greater(t, len(encoding), 32)
}

func assertShortEncoding(t *testing.T, node node.Node) {
	t.Helper()

	encoding := encodeNode(t, node)
	require.LessOrEqual(t, len(encoding), 32)
}
