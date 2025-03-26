// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestAppendMaterializedInPlace(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	length := uint(len(data))

	entry := appendStorageEntry{
		data:          data,
		currentLength: length,
	}

	encoded := entry.value()
	encodedFromLen := scale.MustMarshal(length)
	require.Equal(t, 1, len(encodedFromLen))

	require.True(t, bytes.HasPrefix(encoded, encodedFromLen))
	require.Equal(t, entry.currentLength, length)
	require.Equal(t, *entry.materializedLength, length)
}
