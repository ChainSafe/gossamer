// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsChildStorageKey(t *testing.T) {
	require.True(t, IsChildStorageKey([]byte(":child_storage:default:bleh")))
	require.True(t, IsChildStorageKey([]byte(":child_storage:default:")))

	require.False(t, IsChildStorageKey([]byte("notequal")))
	require.False(t, IsChildStorageKey([]byte("")))
	require.False(t, IsChildStorageKey(nil))
}
