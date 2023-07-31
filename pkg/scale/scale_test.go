// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"reflect"
	"testing"
)

func Test_fieldScaleIndicesCache_fieldScaleIndices(t *testing.T) {
	tests := []struct {
		name        string
		in          interface{}
		wantIndices fieldScaleIndices
		wantErr     bool
	}{
		{
			in: struct{ Foo int }{},
			wantIndices: fieldScaleIndices{
				{
					fieldIndex: 0,
				},
			},
		},
		{
			in: struct {
				End1 bool
				Baz  bool `scale:"3"`
				End2 []byte
				Bar  int32 `scale:"2"`
				End3 []byte
				Foo  []byte `scale:"1"`
			}{},
			wantIndices: fieldScaleIndices{
				{
					fieldIndex: 5,
					scaleIndex: newIntPtr(1),
				},
				{
					fieldIndex: 3,
					scaleIndex: newIntPtr(2),
				},
				{
					fieldIndex: 1,
					scaleIndex: newIntPtr(3),
				},
				{
					fieldIndex: 0,
				},
				{
					fieldIndex: 2,
				},
				{
					fieldIndex: 4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsic := &fieldScaleIndicesCache{
				cache: make(map[string]fieldScaleIndices),
			}
			_, gotIndices, err := fsic.fieldScaleIndices(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("fieldScaleIndicesCache.fieldScaleIndices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIndices, tt.wantIndices) {
				t.Errorf("fieldScaleIndicesCache.fieldScaleIndices() gotIndices = %v, want %v", gotIndices, tt.wantIndices)
			}
		})
	}
}
