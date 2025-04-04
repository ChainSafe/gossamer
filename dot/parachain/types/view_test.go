package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func TestView_ReplaceDifference(t *testing.T) {
	testcases := []struct {
		name  string
		input struct {
			old *View
			new *View
		}
		want []common.Hash
		err  error
	}{
		{
			name: "test1_complete_different",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{{0x01}, {0x02}}},
				new: &View{Heads: []common.Hash{{0x03}, {0x04}}},
			},
			want: []common.Hash{{0x03}, {0x04}},
		},
		{
			name: "test2_partially_different_empty_re",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{{0x01}, {0x02}, {0x03}, {0x04}}},
				new: &View{Heads: []common.Hash{{0x03}, {0x04}}},
			},
			want: []common.Hash{},
		},
		{
			name: "test3_partially_different_non_empty_re",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{{0x01}, {0x02}, {0x03}}},
				new: &View{Heads: []common.Hash{{0x03}, {0x04}}},
			},
			want: []common.Hash{{0x04}},
		},
		{
			name: "test4_partially_different_empty_old",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{}},
				new: &View{Heads: []common.Hash{{0x03}, {0x04}}},
			},
			want: []common.Hash{{0x03}, {0x04}},
		},
		{
			name: "test5_partially_different_empty_new",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{{0x03}, {0x04}}},
				new: &View{Heads: []common.Hash{}},
			},
			want: []common.Hash{},
		},
		{
			name: "test6_partially_different_both_empty",
			input: struct {
				old *View
				new *View
			}{
				old: &View{Heads: []common.Hash{}},
				new: &View{Heads: []common.Hash{}},
			},
			want: []common.Hash{},
		},
	}

	for _, testcase := range testcases {
		oldV := testcase.input.old
		newV := testcase.input.new
		re := oldV.ReplaceDifference(*newV)

		// new view should replace the old view in memory
		assert.Equal(t, newV, oldV)
		// re should be the difference hashes between old and new
		assert.Equal(t, testcase.want, re)
	}
}
