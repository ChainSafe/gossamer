// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsChildStorageKey(t *testing.T) {
	require.True(t, IsDefaultChildStorageKey([]byte(":child_storage:default:bleh")))
	require.True(t, IsDefaultChildStorageKey([]byte(":child_storage:default:")))

	require.False(t, IsDefaultChildStorageKey([]byte("notequal")))
	require.False(t, IsDefaultChildStorageKey([]byte("")))
	require.False(t, IsDefaultChildStorageKey(nil))
}
