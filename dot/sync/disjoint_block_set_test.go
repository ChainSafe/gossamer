// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

func Test_disjointBlockSet_addBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		block            *types.Block
		err              error
	}{
		{
			name: "add block beyond capacity",
			block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			},
			err: errors.New("cannot add block; set is at capacity"),
		},
		{
			name: "add block",
			disjointBlockSet: disjointBlockSet{
				limit:            1,
				blocks:           make(map[common.Hash]*pendingBlock),
				timeNow:          time.Now,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			},
			err: nil,
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addBlock(tt.block)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_disjointBlockSet_addHeader(t *testing.T) {
	t.Parallel()

	mockTime := func() time.Time { return time.Unix(1001, 0) }

	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		header           *types.Header
		err              error
	}{
		{
			name: "add header beyond capactiy",
			header: &types.Header{
				Number: 1,
			},
			err: errors.New("cannot add block; set is at capacity"),
		},
		{
			name: "add header",
			disjointBlockSet: disjointBlockSet{
				blocks:           make(map[common.Hash]*pendingBlock),
				limit:            1,
				timeNow:          mockTime,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			header: &types.Header{
				Number: 1,
			},
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addHeader(tt.header)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_disjointBlockSet_clearBlocks(t *testing.T) {
	t.Parallel()

	testBlock := &pendingBlock{
		clearAt: time.Unix(1000, 0),
		hash:    common.Hash{1},
	}
	testBlock2 := &pendingBlock{
		clearAt: time.Unix(1002, 0),
		hash:    common.Hash{2},
	}
	mockTime := func() time.Time { return time.Unix(1001, 0) }

	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		remaining        int
	}{
		{
			name: "base case",
			disjointBlockSet: disjointBlockSet{
				limit: 0,
				blocks: map[common.Hash]*pendingBlock{
					{1}: testBlock,
				},
				timeNow: mockTime,
			},
			remaining: 0,
		},
		{
			name: "remove clear one block",
			disjointBlockSet: disjointBlockSet{
				limit: 0,
				blocks: map[common.Hash]*pendingBlock{
					{1}: testBlock,
					{2}: testBlock2,
				},
				timeNow: mockTime,
			},
			remaining: 1,
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.disjointBlockSet.clearBlocks()
			assert.Equal(t, tt.remaining, len(tt.disjointBlockSet.blocks))
		})
	}
}

func Test_disjointBlockSet_getBlocks(t *testing.T) {
	t.Parallel()

	testBlock := &pendingBlock{}

	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		want             []*pendingBlock
	}{
		{
			name: "no blocks",
			want: []*pendingBlock{},
		},
		{
			name: "base case",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{}: testBlock,
				},
			},
			want: []*pendingBlock{testBlock},
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &disjointBlockSet{
				blocks: tt.disjointBlockSet.blocks,
			}
			blocks := s.getBlocks()
			assert.Equal(t, tt.want, blocks)
		})
	}
}

func Test_disjointBlockSet_removeLowerBlocks(t *testing.T) {
	t.Parallel()

	testBlock1 := &pendingBlock{
		hash:   common.Hash{1},
		number: 1,
	}
	testBlock10 := &pendingBlock{
		hash:   common.Hash{10},
		number: 10,
	}

	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		num              uint
		remaining        int
	}{
		{
			name: "number 0",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}:  testBlock1,
					{10}: testBlock10,
				},
			},
			num:       0,
			remaining: 2,
		},
		{
			name: "number 1",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}:  testBlock1,
					{10}: testBlock10,
				},
			},
			num:       1,
			remaining: 1,
		},
		{
			name: "number 11",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}:  testBlock1,
					{10}: testBlock10,
				},
			},
			num:       11,
			remaining: 0,
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.disjointBlockSet.removeLowerBlocks(tt.num)
			assert.Equal(t, tt.remaining, len(tt.disjointBlockSet.blocks))
		})
	}
}

func Test_disjointBlockSet_size(t *testing.T) {
	t.Parallel()

	testBlock1 := &pendingBlock{
		hash:   common.Hash{1},
		number: 1,
	}
	testBlock10 := &pendingBlock{
		hash:   common.Hash{10},
		number: 10,
	}
	tests := []struct {
		name             string
		disjointBlockSet disjointBlockSet
		want             int
	}{
		{
			name: "expect 0",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{},
			},
			want: 0,
		},
		{
			name: "expect 1",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					testBlock1.hash: testBlock1,
				},
			},
			want: 1,
		},
		{
			name: "expect 2",
			disjointBlockSet: disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					testBlock1.hash:  testBlock1,
					testBlock10.hash: testBlock10,
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests { //nolint:govet
		tt := tt //nolint:govet
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &disjointBlockSet{
				blocks: tt.disjointBlockSet.blocks,
			}
			assert.Equal(t, tt.want, s.size())
		})
	}
}
