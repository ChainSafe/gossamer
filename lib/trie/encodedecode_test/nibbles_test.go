// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package encodedecode_test

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/trie/decode"
	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/stretchr/testify/assert"
)

func Test_NibblesKeyLE(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		nibblesToEncode []byte
		nibblesDecoded  []byte
	}{
		"empty input": {
			nibblesToEncode: []byte{},
			nibblesDecoded:  []byte{},
		},
		"one byte": {
			nibblesToEncode: []byte{1},
			nibblesDecoded:  []byte{0, 1},
		},
		"two bytes": {
			nibblesToEncode: []byte{1, 2},
			nibblesDecoded:  []byte{1, 2},
		},
		"three bytes": {
			nibblesToEncode: []byte{1, 2, 3},
			nibblesDecoded:  []byte{0, 1, 2, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keyLE := encode.NibblesToKeyLE(testCase.nibblesToEncode)
			nibblesDecoded := decode.KeyLEToNibbles(keyLE)

			assert.Equal(t, testCase.nibblesDecoded, nibblesDecoded)
		})
	}
}
