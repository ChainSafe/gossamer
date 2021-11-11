// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package sync

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveOutlier(t *testing.T) {
	t.Parallel()
	arr := []*big.Int{big.NewInt(100), big.NewInt(-100), big.NewInt(60), big.NewInt(80), big.NewInt(20), big.NewInt(40), big.NewInt(50), big.NewInt(1000)}
	expectedSum := big.NewInt(350) //excluding the outlier -100 and 1000

	sum, count := removeOutlier(arr)
	require.Equal(t, int64(6), count)
	require.Equal(t, expectedSum, sum)
}
