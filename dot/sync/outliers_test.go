// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_removeOutliers(t *testing.T) {
	tests := []struct {
		name      string
		dataArr   []uint
		wantSum   *big.Int
		wantCount uint
	}{
		{
			name:      "base case",
			dataArr:   []uint{100, 0, 260, 280, 220, 240, 250, 1000},
			wantSum:   big.NewInt(1350),
			wantCount: uint(7),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSum, gotCount := removeOutliers(tt.dataArr)
			assert.Equal(t, tt.wantSum, gotSum)
			assert.Equal(t, tt.wantCount, gotCount)
		})
	}
}
