package triedb

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/codec"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb/nibbles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ByteSize(t *testing.T) {
	var childHash hash.H256 = runtime.BlakeTwo256{}.Hash([]byte{0})
	encodedBranch := codec.Branch{
		PartialKey: nibbles.NewNibbles([]byte{1}),
		Value:      codec.InlineValue([]byte{7, 8, 9}),
		Children: [codec.ChildrenCapacity]codec.MerkleValue{
			nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil,
			codec.HashedNode[hash.H256]{Hash: childHash},
		},
	}

	cachedNode, err := NewCachedNodeFromNode[hash.H256, runtime.BlakeTwo256](encodedBranch)
	require.NoError(t, err)
	assert.Equal(t, 308, int(cachedNode.ByteSize()))

	encodedBranch = codec.Branch{
		PartialKey: nibbles.NewNibbles([]byte{1}),
		Value:      codec.InlineValue(bytes.Repeat([]byte{1}, 33)),
		Children: [codec.ChildrenCapacity]codec.MerkleValue{
			nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil,
			codec.HashedNode[hash.H256]{Hash: childHash},
		},
	}
	cachedNode, err = NewCachedNodeFromNode[hash.H256, runtime.BlakeTwo256](encodedBranch)
	require.NoError(t, err)
	assert.Equal(t, 308+30, int(cachedNode.ByteSize()))
}
