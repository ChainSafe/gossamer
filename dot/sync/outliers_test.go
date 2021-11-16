// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"reflect"
	"testing"
)

func Test_getMedian(t *testing.T) {
	type args struct {
		data []*big.Int
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getMedian(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMedian() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeOutliers(t *testing.T) {
	type args struct {
		dataArr []*big.Int
	}
	tests := []struct {
		name      string
		args      args
		wantSum   *big.Int
		wantCount int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSum, gotCount := removeOutliers(tt.args.dataArr)
			if !reflect.DeepEqual(gotSum, tt.wantSum) {
				t.Errorf("removeOutliers() gotSum = %v, want %v", gotSum, tt.wantSum)
			}
			if gotCount != tt.wantCount {
				t.Errorf("removeOutliers() gotCount = %v, want %v", gotCount, tt.wantCount)
			}
		})
	}
}