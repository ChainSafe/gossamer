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
	"sort"
)

//	removeOutlier removes the outlier from the slice
//  Explanation:
// 	IQR outlier detection
// 	Q25 = 25th_percentile
// 	Q75 = 75th_percentile
// 	IQR = Q75 - Q25         // inter-quartile range
// 	If x >  Q75  + 1.5 * IQR or  x   < Q25 - 1.5 * IQR THEN  x is a mild outlier
// 	If x >  Q75  + 3.0 * IQR or  x   < Q25 – 3.0 * IQR THEN  x is a extreme outlier
// Ref: http://www.mathwords.com/o/outlier.htm
//
// returns: sum of all the non-outliers elements
func removeOutlier(dataArr []*big.Int) (*big.Int, int64) {
	length := len(dataArr)

	switch length {
	case 0:
		return big.NewInt(0), 0
	case 1:
		return dataArr[0], 1
	case 2:
		return big.NewInt(0).Add(dataArr[0], dataArr[1]), 2
	}

	//now sort the array
	sort.Slice(dataArr, func(i, j int) bool {
		return dataArr[i].Cmp(dataArr[j]) < 0
	})

	half := length / 2
	data1 := dataArr[:half]
	var data2 []*big.Int

	if length%2 == 0 {
		data2 = dataArr[half:]
	} else {
		data2 = dataArr[half+1:]
	}

	q1 := getMedian(data1)
	q3 := getMedian(data2)

	iqr := big.NewInt(0).Sub(q3, q1)
	iqr1_5 := big.NewInt(0).Mul(iqr, big.NewInt(2)) //instead of 1.5 it is 2.0 due to the rounding
	lower := big.NewInt(0).Sub(q1, iqr1_5)
	upper := big.NewInt(0).Add(q3, iqr1_5)

	reducedValue := big.NewInt(0)
	count := int64(0)
	for _, v := range dataArr {
		//collect valid (non-outlier) values
		lowPass := v.Cmp(lower)
		highPass := v.Cmp(upper)
		if lowPass >= 0 && highPass <= 0 {
			reducedValue = big.NewInt(0).Add(reducedValue, v)
			count++
		}
	}

	return reducedValue, count
}

func getMedian(data []*big.Int) *big.Int {
	length := len(data)
	half := length / 2
	if length%2 == 0 {
		sum := big.NewInt(0).Add(data[half], data[half-1])
		return big.NewInt(0).Div(sum, big.NewInt(2))
	}

	return data[half]
}
