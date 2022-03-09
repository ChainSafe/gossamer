// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveOutlier(t *testing.T) {
	t.Parallel()

	arr := []uint{100, 0, 260, 280, 220, 240, 250, 1000}

	expectedSum := big.NewInt(1350) // excluding the outlier -100 and 1000
	expectedCount := uint(7)

	sum, count := removeOutliers(arr)
	assert.Equal(t, expectedSum, sum)
	assert.Equal(t, expectedCount, count)
}
