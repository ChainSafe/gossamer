// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	randomHashString = "0x580d77a9136035a0bc3c3cd86286172f7f81291164c5914266073a30466fba21"
	emptyHash        = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

func TestCustomUnmarshalJson(t *testing.T) {
	testCases := []struct {
		description string
		hash        string
		errMsg      string
		expected    string
	}{
		{description: "Test empty params", hash: "", errMsg: "invalid hash format"},
		{description: "Test valid params", hash: randomHashString, expected: randomHashString},
		{description: "Test zero hash value", hash: "0x", expected: emptyHash},
		{description: "Test invalid params", hash: "zz", errMsg: "could not byteify non 0x prefixed string"},
	}

	h := Hash{}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			err := h.UnmarshalJSON([]byte(test.hash))
			if test.errMsg != "" {
				require.Equal(t, err.Error(), test.errMsg)
				return
			}
			require.NotNil(t, h)
			require.Equal(t, h.String(), test.expected)
		})
	}
}

func TestCustomMarshalJson(t *testing.T) {
	randomHash, _ := HexToHash(randomHashString)
	testCases := []struct {
		description string
		hash        Hash
		expected    string
	}{
		{description: "Test empty params", hash: Hash{}, expected: emptyHash},
		{description: "Test valid params", hash: randomHash, expected: randomHashString},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			byt, err := test.hash.MarshalJSON()
			require.Nil(t, err)
			require.True(t, strings.Contains(string(byt), test.expected))
		})
	}
}

func Test_Hash_IsEmpty(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		hash  Hash
		empty bool
	}{
		"empty": {
			empty: true,
		},
		"not empty": {
			hash: Hash{1},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			empty := testCase.hash.IsEmpty()

			assert.Equal(t, testCase.empty, empty)
		})
	}
}

func Benchmark_IsEmpty(b *testing.B) {
	h := Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b.Run("using equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = h == Hash{}
		}
	})

	b.Run("using equal with predefined empty", func(b *testing.B) {
		empty := Hash{}
		for i := 0; i < b.N; i++ {
			_ = h == empty
		}
	})

	b.Run("using bytes.Equal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = h.Equal(Hash{})
		}
	})
}
