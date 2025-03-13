// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyExtrinsicsInto(t *testing.T) {
	set := map[uint32]struct{}{}

	ext := Extrinsics{1, 2, 3}
	ext.CopyExtrinsicsInto(set)

	require.Equal(t, 3, len(set))

	for _, ext := range ext {
		require.NotNil(t, set[ext])
	}
}

func TestInsertExtrinsic(t *testing.T) {
	ext := Extrinsics{1, 2, 3}
	ext.Insert(4)

	require.Equal(t, 4, len(ext))
	require.Equal(t, uint32(4), ext[3])
}

func TestExtendExtrinsic(t *testing.T) {
	ext := Extrinsics{1, 2, 3}
	other := Extrinsics{4, 5, 6}

	ext.Extend(other)

	require.Equal(t, 6, len(ext))

	for i := 1; i <= 6; i++ {
		require.Equal(t, uint32(i), ext[i-1])
	}
}
