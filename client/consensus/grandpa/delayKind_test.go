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
	_, isFinalizedType := delayKind.value.(Finalized)
	require.True(t, isFinalizedType)

	medLastFinalized := uint(3)
	bestKind := Best{medianLastFinalized: medLastFinalized}
	delayKind = newDelayKind(bestKind)
	best, isBestType := delayKind.value.(Best)
	require.True(t, isBestType)
	require.Equal(t, medLastFinalized, best.medianLastFinalized)
}
