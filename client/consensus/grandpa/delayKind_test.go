// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDelayKind(t *testing.T) {
	finalizedKind := Finalized{}
	delayKind := newDelayKind[uint](finalizedKind)
	_, isFinalizedType := delayKind.value.(Finalized)
	require.True(t, isFinalizedType)

	medLastFinalized := uint(3)
	bestKind := Best[uint]{medianLastFinalized: medLastFinalized}
	delayKind = newDelayKind[uint](bestKind)
	best, isBestType := delayKind.value.(Best[uint])
	require.True(t, isBestType)
	require.Equal(t, medLastFinalized, best.medianLastFinalized)
}
