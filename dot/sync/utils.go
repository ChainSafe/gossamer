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

// IQR golang outlier detection
// 	Q25 = 25th_percentile
// 	Q75 = 75th_percentile
// 	IQR = Q75 - Q25         // inter-quartile range
// 	If x >  Q75  + 1.5 * IQR or  x   < Q25 - 1.5 * IQR THEN  x is a mild outlier
// 	If x >  Q75  + 3.0 * IQR or  x   < Q25 – 3.0 * IQR THEN  x is a extreme outlier
// Ref: http://www.mathwords.com/o/outlier.htm
func outlierDetection(arr []int64) (int, int) {
	if len(arr) < 3 {
		return -1, -1
	}

	q1 := arr[0]
	q3 := arr[len(arr)-1]

	if q1 == q3 {
		return -1, -1
	}

	iqr := float64(q3 - q1)
	iqr1_5 := iqr * 1.5
	lower := float64(q1) - iqr1_5
	upper := float64(q3) + iqr1_5

	lowerIndex := -1
	upperIndex := -1

	for i, v := range arr {
		v := float64(v)
		if v >= lower && lowerIndex == -1 {
			lowerIndex = i
		}

		if v <= upper && upperIndex == -1 {
			upperIndex = i
		}
	}

	return lowerIndex, upperIndex
}

func getMedian(data []int64) int64 {
	len:=len(data)
	half := len/2
	if ( len % 2 == 0)
		return data[half] + data[half-1] / 2;
	else
		return data.get(data.size() / 2);
}
