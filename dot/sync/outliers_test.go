// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveOutlier(t *testing.T) {
	t.Parallel()
	arr := []*big.Int{
		big.NewInt(100), big.NewInt(-100), big.NewInt(60),
		big.NewInt(80), big.NewInt(20), big.NewInt(40),
		big.NewInt(50), big.NewInt(1000),
	}
	expectedSum := big.NewInt(350) // excluding the outlier -100 and 1000

	sum, count := removeOutliers(arr)
	require.Equal(t, int64(6), count)
	require.Equal(t, expectedSum, sum)
}
