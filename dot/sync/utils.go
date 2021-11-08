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

type reducer func(prevValue interface{}, newValue interface{}) interface{}

// for comp: comp(x, y) = -1: x<y, 0: x==y, 1: x>y
type comperator func(prevValue interface{}, newValue interface{}) int

//RemoveOutlier removes the outlier from the slice
//  Explanation:
// 	IQR outlier detection
// 	Q25 = 25th_percentile
// 	Q75 = 75th_percentile
// 	IQR = Q75 - Q25         // inter-quartile range
// 	If x >  Q75  + 1.5 * IQR or  x   < Q25 - 1.5 * IQR THEN  x is a mild outlier
// 	If x >  Q75  + 3.0 * IQR or  x   < Q25 – 3.0 * IQR THEN  x is a extreme outlier
// Ref: http://www.mathwords.com/o/outlier.htm
// returns: reducer output
func RemoveOutlier(sortedArr []interface{}, compFn comperator, initialReducedVal interface{}, reducer, plusFn, minusFn, divideFn, multiplyFn reducer) interface{} {
	length := len(sortedArr)

	switch length {
	case 0:
		return nil
	case 1:
		return sortedArr[0]
	case 2:
		return reducer(sortedArr[0], sortedArr[1])
	}

	half := length / 2
	data1 := sortedArr[:half]
	var data2 []interface{}

	if length%2 == 0 {
		data2 = sortedArr[half:]
	} else {
		data2 = sortedArr[half+1:]
	}

	q1 := getMedian(data1, plusFn, divideFn)
	q3 := getMedian(data2, plusFn, divideFn)

	iqr := minusFn(q3, q1)
	iqr1_5 := multiplyFn(iqr, 1.5)
	lower := minusFn(q1, iqr1_5)
	upper := plusFn(q3, iqr1_5)

	reducedValue := initialReducedVal
	for _, v := range sortedArr {
		//collect valid (non-outlier) values
		lowPass := compFn(v, lower)
		highPass := compFn(v, upper)
		if lowPass >= 0 && highPass <= 0 {
			reducedValue = reducer(reducedValue, v)
		}
	}

	return reducedValue
}

func getMedian(data []interface{}, sum, divide reducer) interface{} {
	length := len(data)
	half := length / 2
	if length%2 == 0 {
		return divide(sum(data[half], data[half-1]), 2)
	}

	return data[half]
}
