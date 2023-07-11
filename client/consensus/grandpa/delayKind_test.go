// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDelayKind(t *testing.T) {
	finalizedKind := Finalized{}
	delayKind := newDelayKind(finalizedKind)
	_, isFinalizedType := delayKind.Value.(Finalized)
	require.True(t, isFinalizedType)

	medLastFinalized := uint(3)
	bestKind := Best{MedianLastFinalized: medLastFinalized}
	delayKind = newDelayKind(bestKind)
	best, isBestType := delayKind.Value.(Best)
	require.True(t, isBestType)
	require.Equal(t, medLastFinalized, best.MedianLastFinalized)
}
