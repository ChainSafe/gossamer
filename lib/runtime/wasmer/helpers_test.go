// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"encoding/json"
	"errors"
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
