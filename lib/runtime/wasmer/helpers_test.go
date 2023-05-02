// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genesisFromRawJSON(t *testing.T, jsonFilepath string) (gen genesis.Genesis) {
	t.Helper()

	fp, err := filepath.Abs(jsonFilepath)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Clean(fp))
	require.NoError(t, err)

	err = json.Unmarshal(data, &gen)
	require.NoError(t, err)

	return gen
}

func TestMemory_safeCastInt32(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		value     uint32
		exp       int32
		expErr    error
		expErrMsg string
	}{
		{
			name:  "valid cast",
			value: uint32(0),
			exp:   int32(0),
		},
		{
			name:  "max uint32",
			value: uint32(math.MaxInt32),
			exp:   math.MaxInt32,
		},
		{
			name:      "out of bounds",
			value:     uint32(math.MaxInt32 + 1),
			expErr:    errMemoryValueOutOfBounds,
			expErrMsg: errMemoryValueOutOfBounds.Error(),
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			res, err := safeCastInt32(test.value)
			assert.ErrorIs(t, err, test.expErr)
			if test.expErr != nil {
				assert.EqualError(t, err, test.expErrMsg)
			}
			assert.Equal(t, test.exp, res)
		})
	}
}

func Test_pointerSize(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		ptr         uint32
		size        uint32
		pointerSize int64
	}{
		"0": {},
		"ptr_8_size_32": {
			ptr:         8,
			size:        32,
			pointerSize: int64(8) | (int64(32) << 32),
		},
		"ptr_max_uint32_and_size_max_uint32": {
			ptr:         ^uint32(0),
			size:        ^uint32(0),
			pointerSize: ^int64(0),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pointerSize := toPointerSize(testCase.ptr, testCase.size)

			require.Equal(t, testCase.pointerSize, pointerSize)

			ptr, size := splitPointerSize(pointerSize)

			assert.Equal(t, testCase.ptr, ptr)
			assert.Equal(t, testCase.size, size)
		})
	}
}

func Test_panicOnError(t *testing.T) {
	t.Parallel()

	err := (error)(nil)
	assert.NotPanics(t, func() { panicOnError(err) })

	err = errors.New("test error")
	assert.PanicsWithValue(t, err, func() { panicOnError(err) })
}
