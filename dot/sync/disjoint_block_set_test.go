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

	hashHeader := func(header types.Header) common.Hash {
		return header.Hash()
	}
	setHashToHeader := func(header types.Header) *types.Header {
		header.Hash()
		return &header
	}

	timeNow := func() time.Time {
		return time.Unix(0, 0)
	}
	tests := map[string]struct {
		disjointBlockSet         *disjointBlockSet
		block                    *types.Block
		expectedDisjointBlockSet *disjointBlockSet
		err                      error
	}{
		"add block beyond capacity": {
			disjointBlockSet: &disjointBlockSet{},
			block: &types.Block{
				Header: types.Header{
					Number: 1,
				},
			},
			expectedDisjointBlockSet: &disjointBlockSet{},
			err:                      errSetAtLimit,
		},
		"add block": {
			disjointBlockSet: &disjointBlockSet{
				limit:            1,
				blocks:           make(map[common.Hash]*pendingBlock),
				timeNow:          timeNow,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			block: &types.Block{
				Header: types.Header{
					Number:     1,
					ParentHash: common.Hash{1},
				},
				Body: []types.Extrinsic{[]byte{1}},
			},
			expectedDisjointBlockSet: &disjointBlockSet{
				limit: 1,
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:    hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:  1,
						header:  setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						body:    &types.Body{{1}},
						clearAt: time.Unix(0, int64(ttl)),
					},
				},
				parentToChildren: map[common.Hash]map[common.Hash]struct{}{
					{1}: {
						hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {},
					},
				},
			},
		},
		"has block": {
			disjointBlockSet: &disjointBlockSet{
				limit: 1,
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:    hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:  1,
						header:  setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						body:    &types.Body{{1}},
						clearAt: time.Unix(0, int64(ttl)),
					},
				},
				timeNow:          timeNow,
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			block: &types.Block{
				Header: types.Header{
					Number:     1,
					ParentHash: common.Hash{1},
				},
				Body: []types.Extrinsic{[]byte{1}},
			},
			expectedDisjointBlockSet: &disjointBlockSet{
				limit: 1,
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:          hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:        1,
						header:        setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						body:          &types.Body{{1}},
						justification: nil,
						clearAt:       time.Unix(0, int64(ttl)),
					},
				},
				parentToChildren: map[common.Hash]map[common.Hash]struct{}{},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addBlock(tt.block)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			tt.disjointBlockSet.timeNow = nil
			assert.Equal(t, tt.expectedDisjointBlockSet, tt.disjointBlockSet)
		})
	}
}

func Test_disjointBlockSet_addHeader(t *testing.T) {
	t.Parallel()

	hashHeader := func(header types.Header) common.Hash {
		return header.Hash()
	}
	setHashToHeader := func(header types.Header) *types.Header {
		header.Hash()
		return &header
	}

	tests := map[string]struct {
		disjointBlockSet         *disjointBlockSet
		header                   *types.Header
		expectedDisjointBlockSet *disjointBlockSet
		err                      error
	}{
		"add header beyond capactiy": {
			disjointBlockSet: &disjointBlockSet{},
			header: &types.Header{
				Number: 1,
			},
			expectedDisjointBlockSet: &disjointBlockSet{},
			err:                      errors.New("cannot add block; set is at capacity"),
		},
		"add header": {
			disjointBlockSet: &disjointBlockSet{
				blocks:           make(map[common.Hash]*pendingBlock),
				limit:            1,
				timeNow:          func() time.Time { return time.Unix(0, 0) },
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			header: &types.Header{
				Number:     1,
				ParentHash: common.Hash{1},
			},
			expectedDisjointBlockSet: &disjointBlockSet{
				limit: 1,
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:    hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:  1,
						header:  setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						clearAt: time.Unix(0, int64(ttl)),
					},
				},
				parentToChildren: map[common.Hash]map[common.Hash]struct{}{
					{1}: {
						hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {},
					},
				},
			},
		},
		"has header": {
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:    hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:  1,
						header:  setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						clearAt: time.Unix(0, int64(ttl)),
					},
				},
				limit:            1,
				timeNow:          func() time.Time { return time.Unix(0, 0) },
				parentToChildren: make(map[common.Hash]map[common.Hash]struct{}),
			},
			header: &types.Header{
				Number:     1,
				ParentHash: common.Hash{1},
			},
			expectedDisjointBlockSet: &disjointBlockSet{
				limit: 1,
				blocks: map[common.Hash]*pendingBlock{
					hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}): {
						hash:    hashHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						number:  1,
						header:  setHashToHeader(types.Header{Number: 1, ParentHash: common.Hash{1}}),
						clearAt: time.Unix(0, int64(ttl)),
					},
				},
				parentToChildren: map[common.Hash]map[common.Hash]struct{}{},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tt.disjointBlockSet.addHeader(tt.header)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			tt.disjointBlockSet.timeNow = nil
			assert.Equal(t, tt.expectedDisjointBlockSet, tt.disjointBlockSet)
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
		name                 string
		disjointBlockSet     *disjointBlockSet
		want                 []*pendingBlock
		wantDisjointBlockSet *disjointBlockSet
	}{
		{
			name:                 "no blocks",
			disjointBlockSet:     &disjointBlockSet{},
			want:                 []*pendingBlock{},
			wantDisjointBlockSet: &disjointBlockSet{},
		},
		{
			name: "base case",
			disjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{}: {},
				},
			},
			want: []*pendingBlock{{}},
			wantDisjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{}: {},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocks := tt.disjointBlockSet.getBlocks()
			assert.Equal(t, tt.want, blocks)
			assert.Equal(t, tt.wantDisjointBlockSet, tt.disjointBlockSet)
		})
	}
}

func Test_disjointBlockSet_removeLowerBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		disjointBlockSet     *disjointBlockSet
		num                  uint
		remaining            map[common.Hash]*pendingBlock
		wantDisjointBlockSet *disjointBlockSet
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
			wantDisjointBlockSet: &disjointBlockSet{
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
			wantDisjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{
					{10}: {
						hash:   common.Hash{10},
						number: 10,
					},
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
			wantDisjointBlockSet: &disjointBlockSet{
				blocks: map[common.Hash]*pendingBlock{},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.disjointBlockSet.removeLowerBlocks(tt.num)
			assert.Equal(t, tt.remaining, tt.disjointBlockSet.blocks)
			assert.Equal(t, tt.wantDisjointBlockSet, tt.disjointBlockSet)
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
			size := tt.disjointBlockSet.size()
			assert.Equal(t, tt.want, size)
		})
	}
}
