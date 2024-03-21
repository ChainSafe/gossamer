package inmemory

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/assert"
)

func Test_Version_Root(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version  trie.TrieLayout
		trie     trie.Trie
		entries  trie.Entries
		expected common.Hash
	}{
		"v0": {
			version: trie.V0,
			entries: trie.Entries{
				trie.Entry{Key: []byte("key1"), Value: []byte("value1")},
				trie.Entry{Key: []byte("key2"), Value: []byte("value2")},
				trie.Entry{Key: []byte("key3"), Value: []byte("verylargevaluewithmorethan32byteslength")},
			},
			expected: common.Hash{
				0x71, 0x5, 0x2d, 0x48, 0x70, 0x46, 0x58, 0xa8, 0x43, 0x5f, 0xb9, 0xcb, 0xc7, 0xef, 0x69, 0xc7, 0x5d,
				0xad, 0x2f, 0x64, 0x0, 0x1c, 0xb3, 0xb, 0xfa, 0x1, 0xf, 0x7d, 0x60, 0x9e, 0x26, 0x57,
			},
		},
		"v1": {
			version: trie.V1,
			entries: trie.Entries{
				trie.Entry{Key: []byte("key1"), Value: []byte("value1")},
				trie.Entry{Key: []byte("key2"), Value: []byte("value2")},
				trie.Entry{Key: []byte("key3"), Value: []byte("verylargevaluewithmorethan32byteslength")},
			},
			expected: common.Hash{
				0x6a, 0x4a, 0x73, 0x27, 0x57, 0x26, 0x3b, 0xf2, 0xbc, 0x4e, 0x3, 0xa3, 0x41, 0xe3, 0xf8, 0xea, 0x63,
				0x5f, 0x78, 0x99, 0x6e, 0xc0, 0x6a, 0x6a, 0x96, 0x5d, 0x50, 0x97, 0xa2, 0x91, 0x1c, 0x29,
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			maxInline, err := testCase.version.Root(NewEmptyInmemoryTrie(), testCase.entries)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, maxInline)
		})
	}
}
