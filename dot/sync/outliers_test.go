// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_nonOutliersSumCount(t *testing.T) {
	tests := []struct {
		name      string
		dataArr   []uint
		wantSum   *big.Int
		wantCount uint
	}{
		{
			name:      "case 0 outliers",
			dataArr:   []uint{2, 5, 6, 9, 12},
			wantSum:   big.NewInt(34),
			wantCount: uint(5),
		},
		{
			name:      "case 1 outliers",
			dataArr:   []uint{100, 2, 260, 280, 220, 240, 250, 1000},
			wantSum:   big.NewInt(1352),
			wantCount: uint(7),
		},
		{
			name:      "case 2 outliers",
			dataArr:   []uint{5000, 500, 5560, 5580, 5520, 5540, 5550, 100000},
			wantSum:   big.NewInt(32750),
			wantCount: uint(6),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSum, gotCount := nonOutliersSumCount(tt.dataArr)
			assert.Equal(t, tt.wantSum, gotSum)
			assert.Equal(t, tt.wantCount, gotCount)
		})
	}
}
