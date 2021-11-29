// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeRoot(t *testing.T) {
	trie := NewEmptyTrie()

	for i := 0; i < 20; i++ {
		rt := GenerateRandomTests(t, 16)
		for _, test := range rt {
			trie.Put(test.key, test.value)

			val := trie.Get(test.key)
			if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get Key %x with value %x: got %x", test.Key(), test.value, val)
			}

			buffer := bytes.NewBuffer(nil)
			err := trie.root.Encode(buffer)
			require.NoError(t, err)
		}
	}
}
