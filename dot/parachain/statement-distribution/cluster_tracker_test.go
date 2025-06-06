// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
)

func TestNewClusterTracker(t *testing.T) {
	clusterTracker := newClusterTracker([]parachaintypes.ValidatorIndex{4, 6, 3, 9}, 3)
	require.NotNil(t, clusterTracker)
}
