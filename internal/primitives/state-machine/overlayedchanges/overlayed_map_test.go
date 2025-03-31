// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloneOverlayedMap(t *testing.T) {
	overlayed := NewOverlayedMap[string, string, *GenericOverlayedEntry[string]]()
	key := "key1"
	overlayed.SetOffchain(key, "value1", nil)

	cloned := overlayed.Clone()

	overlayedEntry, has := overlayed.Get(key)
	require.True(t, has)

	clonedEntry, has := cloned.Get(key)
	require.True(t, has)

	require.Equal(t, overlayedEntry, clonedEntry)

	cloned.SetOffchain(key, "value2", nil)

	overlayedEntry, has = overlayed.Get(key)
	require.True(t, has)

	clonedEntry, has = cloned.Get(key)
	require.True(t, has)

	require.NotEqual(t, overlayedEntry, clonedEntry)
}
