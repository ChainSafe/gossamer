// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewTrieFromGenesis(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		genesis     genesis.Genesis
		expectedKV  map[string]string
		errSentinel error
		errMessage  string
	}{
		"genesis_top_not_found": {
			genesis:     genesis.Genesis{Name: "genesis_name"},
			errSentinel: ErrGenesisTopNotFound,
			errMessage:  "genesis top not found: in genesis genesis_name",
		},
		"bad_hex_trie_key": {
			genesis: genesis.Genesis{
				Name: "genesis_name",
				Genesis: genesis.Fields{
					Raw: map[string]map[string]string{
						"top": {
							"badhexkey": "0xa",
						},
					},
				},
			},
			errSentinel: common.ErrNoPrefix,
			errMessage: "loading genesis top key values into trie: " +
				"cannot convert key hex to bytes: " +
				"could not byteify non 0x prefixed string: badhexkey",
		},
		"success": {
			genesis: genesis.Genesis{
				Name: "genesis_name",
				Genesis: genesis.Fields{
					Raw: map[string]map[string]string{
						"top": {
							"0x0102": "0x0a",
							"0x0103": "0x0b",
						},
					},
				},
			},
			expectedKV: map[string]string{
				"0x0102": "0x0a",
				"0x0103": "0x0b",
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tr, err := NewInMemoryTrieFromGenesis(testCase.genesis)

			require.ErrorIs(t, err, testCase.errSentinel)
			if testCase.errSentinel != nil {
				assert.EqualError(t, err, testCase.errMessage)
				return
			}

			for hexKey, hexValue := range testCase.expectedKV {
				key := common.MustHexToBytes(hexKey)
				value := tr.Get(key)
				assert.Equal(t, hexValue, common.BytesToHex(value))
				tr.Delete(key)
			}
			entries := tr.Entries()
			assert.Empty(t, entries)
		})
	}
}
