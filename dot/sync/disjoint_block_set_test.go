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
		disjointBlockSet *disjointBlockSet
		block            *types.Block
		err              error
	}{
		{
			name:             "add block beyond capacity",
			disjointBlockSet: &disjointBlockSet{},
			block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			},
			err: errSetAtLimit,
		},
		{
			name: "add block",
			disjointBlockSet: &disjointBlockSet{
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
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addBlock(tt.block)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tt.disjointBlockSet.blocks[tt.block.Header.Hash()])
			}
		})
	}
}

func Test_disjointBlockSet_addHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		disjointBlockSet *disjointBlockSet
		header           *types.Header
		err              error
	}{
		{
			name:             "add header beyond capactiy",
			disjointBlockSet: &disjointBlockSet{},
			header: &types.Header{
				Number: 1,
			},
			err: errors.New("cannot add block; set is at capacity"),
		},
		{
			name: "add header",
			disjointBlockSet: &disjointBlockSet{
				blocks:           make(map[common.Hash]*pendingBlock),
				limit:            1,
				timeNow:          func() time.Time { return time.Unix(1001, 0) },
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			header: &types.Header{
				Number: 1,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addHeader(tt.header)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tt.disjointBlockSet.blocks[tt.header.Hash()])
			}
		})
	}
}

func Test_disjointBlockSet_clearBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		disjointBlockSet *disjointBlockSet
		remaining        map[common.Hash]*pendingBlock
	}{
		{
			name: "base case",
			disjointBlockSet: &disjointBlockSet{
				limit: 0,
				blocks: map[common.Hash]*pendingBlock{
					{1}: {
						clearAt: time.Unix(1000, 0),
						hash:    common.Hash{1},
					},
				},
				timeNow: func() time.Time { return time.Unix(1001, 0) },
			},
			remaining: map[common.Hash]*pendingBlock{},
		},
		{
			name: "remove clear one block",
			disjointBlockSet: &disjointBlockSet{
				limit: 0,
				blocks: map[common.Hash]*pendingBlock{
					{1}: {
						clearAt: time.Unix(1000, 0),
						hash:    common.Hash{1},
					},
					{2}: {
						clearAt: time.Unix(1002, 0),
						hash:    common.Hash{2},
					},
				},
				timeNow: func() time.Time { return time.Unix(1001, 0) },
			},
			remaining: map[common.Hash]*pendingBlock{
				{2}: {
					clearAt: time.Unix(1002, 0),
					hash:    common.Hash{2},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.disjointBlockSet.clearBlocks()
			assert.Equal(t, tt.remaining, tt.disjointBlockSet.blocks)
		})
	}
}

func Test_disjointBlockSet_getBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		disjointBlockSet *disjointBlockSet
		want             []*pendingBlock
	}{
		{
			name:             "no blocks",
			disjointBlockSet: &disjointBlockSet{},
			want:             []*pendingBlock{},
		},
		{
			name: "base case",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{}: {},
				},
			},
			want: []*pendingBlock{{}},
		},
	}
	for _, tt := range tests {
		tt := tt
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

	tests := []struct {
		name             string
		disjointBlockSet *disjointBlockSet
		num              uint
		remaining        map[common.Hash]*pendingBlock
	}{
		{
			name: "number 0",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}: {
						hash:   common.Hash{1},
						number: 1,
					},
					{10}: {
						hash:   common.Hash{10},
						number: 10,
					},
				},
			},
			num: 0,
			remaining: map[common.Hash]*pendingBlock{
				{1}: {
					hash:   common.Hash{1},
					number: 1,
				},
				{10}: {
					hash:   common.Hash{10},
					number: 10,
				},
			},
		},
		{
			name: "number 1",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}: {
						hash:   common.Hash{1},
						number: 1,
					},
					{10}: {
						hash:   common.Hash{10},
						number: 10,
					},
				},
			},
			num: 1,
			remaining: map[common.Hash]*pendingBlock{{10}: {
				hash:   common.Hash{10},
				number: 10,
			},
			},
		},
		{
			name: "number 11",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}: {
						hash:   common.Hash{1},
						number: 1,
					},
					{10}: {
						hash:   common.Hash{10},
						number: 10,
					},
				},
			},
			num:       11,
			remaining: map[common.Hash]*pendingBlock{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.disjointBlockSet.removeLowerBlocks(tt.num)
			assert.Equal(t, tt.remaining, tt.disjointBlockSet.blocks)
		})
	}
}

func Test_disjointBlockSet_size(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		disjointBlockSet *disjointBlockSet
		want             int
	}{
		{
			name: "expect 0",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{},
			},
			want: 0,
		},
		{
			name: "expect 1",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}: {hash: common.Hash{1}, number: 1},
				},
			},
			want: 1,
		},
		{
			name: "expect 2",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{1}:  {hash: common.Hash{1}, number: 1},
					{10}: {hash: common.Hash{10}, number: 10},
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := tt.disjointBlockSet
			size := s.size()
			assert.Equal(t, tt.want, size)
		})
	}
}
