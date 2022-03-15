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

var (
	testBlock1 = &pendingBlock{
		hash:   common.Hash{1},
		number: 1,
	}
	testBlock10 = &pendingBlock{
		hash:   common.Hash{10},
		number: 10,
	}
)

func Test_disjointBlockSet_addBlock(t *testing.T) {
	type fields struct {
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
		timeNow          func() time.Time
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name:   "add block beyond capacity",
			fields: fields{},
			args: args{block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			}},
			err: errors.New("cannot add block; set is at capacity"),
		},
		{
			name: "add block",
			fields: fields{
				limit:            1,
				blocks:           make(map[common.Hash]*pendingBlock),
				timeNow:          time.Now,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			args: args{block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			}},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
				timeNow:          tt.fields.timeNow,
			}
			err := s.addBlock(tt.args.block)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_disjointBlockSet_addHeader(t *testing.T) {
	type fields struct {
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
		timeNow          func() time.Time
	}
	type args struct {
		header *types.Header
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		err    error
	}{
		{
			name:   "add header beyond capactiy",
			fields: fields{},
			args: args{header: &types.Header{
				Number: 1,
			}},
			err: errors.New("cannot add block; set is at capacity"),
		},
		{
			name: "add header",
			fields: fields{
				blocks:           make(map[common.Hash]*pendingBlock),
				limit:            1,
				timeNow:          time.Now,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			args: args{header: &types.Header{
				Number: 1,
			}},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
				timeNow:          tt.fields.timeNow,
			}
			err := s.addHeader(tt.args.header)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_disjointBlockSet_clearBlocks(t *testing.T) {
	testBlock := &pendingBlock{
		clearAt: time.Now(),
	}
	type fields struct {
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
		timeNow          func() time.Time
	}
	tests := []struct {
		name      string
		fields    fields
		remaining int
	}{
		{
			name: "base case",
			fields: fields{
				limit: 0,
				blocks: map[common.Hash]*pendingBlock{
					common.Hash{}: testBlock, //nolint:gofmt
				},
				timeNow: time.Now,
			},
			remaining: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
				timeNow:          tt.fields.timeNow,
			}
			s.clearBlocks()
			assert.Equal(t, tt.remaining, len(tt.fields.blocks))
		})
	}
}

func Test_disjointBlockSet_getBlocks(t *testing.T) {
	testBlock := &pendingBlock{}
	type fields struct {
		blocks map[common.Hash]*pendingBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   []*pendingBlock
	}{
		{
			name:   "no blocks",
			fields: fields{},
			want:   []*pendingBlock{},
		},
		{
			name: "base case",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					common.Hash{}: testBlock, //nolint:gofmt
				},
			},
			want: []*pendingBlock{testBlock},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				blocks: tt.fields.blocks,
			}
			blocks := s.getBlocks()
			assert.Equalf(t, tt.want, blocks, "getBlocks()")
		})
	}
}

func Test_disjointBlockSet_removeLowerBlocks(t *testing.T) {
	t.Parallel()
	type fields struct {
		blocks map[common.Hash]*pendingBlock
	}

	tests := []struct {
		name      string
		fields    fields
		num       uint
		remaining int
	}{
		{
			name: "number 0",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					common.Hash{1}:  testBlock1, //nolint:gofmt
					common.Hash{10}: testBlock10,
				},
			},
			num:       0,
			remaining: 2,
		},
		{
			name: "number 1",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					common.Hash{1}:  testBlock1, //nolint:gofmt
					common.Hash{10}: testBlock10,
				},
			},
			num:       1,
			remaining: 1,
		},
		{
			name: "number 11",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					common.Hash{1}:  testBlock1, //nolint:gofmt
					common.Hash{10}: testBlock10,
				},
			},
			num:       11,
			remaining: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				blocks: tt.fields.blocks,
			}
			s.removeLowerBlocks(tt.num)
			assert.Equal(t, tt.remaining, len(s.blocks))
		})
	}
}

func Test_disjointBlockSet_size(t *testing.T) {
	t.Parallel()

	type fields struct {
		blocks map[common.Hash]*pendingBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "expect 0",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{},
			},
			want: 0,
		},
		{
			name: "expect 1",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					testBlock1.hash: testBlock1,
				},
			},
			want: 1,
		},
		{
			name: "expect 2",
			fields: fields{
				blocks: map[common.Hash]*pendingBlock{
					testBlock1.hash:  testBlock1,
					testBlock10.hash: testBlock10,
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				blocks: tt.fields.blocks,
			}
			assert.Equalf(t, tt.want, s.size(), "size()")
		})
	}
}
