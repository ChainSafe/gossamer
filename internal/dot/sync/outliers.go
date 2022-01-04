// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"sort"
)

// removeOutliers removes the outlier from the slice
// Explanation:
// IQR outlier detection
// Q25 = 25th_percentile
// Q75 = 75th_percentile
// IQR = Q75 - Q25         // inter-quartile range
// If x >  Q75  + 1.5 * IQR or  x   < Q25 - 1.5 * IQR THEN  x is a mild outlier
// If x >  Q75  + 3.0 * IQR or  x   < Q25 – 3.0 * IQR THEN  x is a extreme outlier
// Ref: http://www.mathwords.com/o/outlier.htm
//
// returns: sum of all the non-outliers elements
func removeOutliers(dataArr []*big.Int) (sum *big.Int, count int64) {
	length := len(dataArr)

	switch length {
	case 0:
		return big.NewInt(0), 0
	case 1:
		return dataArr[0], 1
	case 2:
		return big.NewInt(0).Add(dataArr[0], dataArr[1]), 2
	}

	sort.Slice(dataArr, func(i, j int) bool {
		return dataArr[i].Cmp(dataArr[j]) < 0
	})

	half := length / 2
	firstHalf := dataArr[:half]
	var secondHalf []*big.Int

	if length%2 == 0 {
		secondHalf = dataArr[half:]
	} else {
		secondHalf = dataArr[half+1:]
	}

	q1 := getMedian(firstHalf)
	q3 := getMedian(secondHalf)

	iqr := big.NewInt(0).Sub(q3, q1)
	iqr1_5 := big.NewInt(0).Mul(iqr, big.NewInt(2)) // instead of 1.5 it is 2.0 due to the rounding
	lower := big.NewInt(0).Sub(q1, iqr1_5)
	upper := big.NewInt(0).Add(q3, iqr1_5)

	sum = big.NewInt(0)
	count = int64(0)
	for _, v := range dataArr {
		// collect valid (non-outlier) values
		lowPass := v.Cmp(lower)
		highPass := v.Cmp(upper)
		if lowPass >= 0 && highPass <= 0 {
			sum.Add(sum, v)
			count++
		}
	}

	return sum, count
}

func getMedian(data []*big.Int) *big.Int {
	length := len(data)
	half := length / 2
	if length%2 == 0 {
		sum := big.NewInt(0).Add(data[half], data[half-1])
		return sum.Div(sum, big.NewInt(2))
	}

	return data[half]
}
