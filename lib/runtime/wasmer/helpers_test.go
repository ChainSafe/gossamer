// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pointerSize(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		ptr         uint32
		size        uint32
		pointerSize int64
	}{
		"0": {},
		"ptr 8 size 32": {
			ptr:         8,
			size:        32,
			pointerSize: int64(8) | (int64(32) << 32),
		},
		"ptr max uint32 and size max uint32": {
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
