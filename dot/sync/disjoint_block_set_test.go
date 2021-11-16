// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"math/big"
	"reflect"
	"sync"
	"testing"
)

func Test_disjointBlockSet_addBlock(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		block *types.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if err := s.addBlock(tt.args.block); (err != nil) != tt.wantErr {
				t.Errorf("addBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_disjointBlockSet_addHashAndNumber(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash   common.Hash
		number *big.Int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if err := s.addHashAndNumber(tt.args.hash, tt.args.number); (err != nil) != tt.wantErr {
				t.Errorf("addHashAndNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_disjointBlockSet_addHeader(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		header *types.Header
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if err := s.addHeader(tt.args.header); (err != nil) != tt.wantErr {
				t.Errorf("addHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_disjointBlockSet_addJustification(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash common.Hash
		just []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if err := s.addJustification(tt.args.hash, tt.args.just); (err != nil) != tt.wantErr {
				t.Errorf("addJustification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_disjointBlockSet_addToParentMap(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		parent common.Hash
		child  common.Hash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_disjointBlockSet_getBlock(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash common.Hash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *pendingBlock
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.getBlock(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disjointBlockSet_getBlocks(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		want   []*pendingBlock
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.getBlocks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disjointBlockSet_getChildren(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash common.Hash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[common.Hash]struct{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.getChildren(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getChildren() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disjointBlockSet_getReadyDescendants(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		curr  common.Hash
		ready []*types.BlockData
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*types.BlockData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.getReadyDescendants(tt.args.curr, tt.args.ready); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getReadyDescendants() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disjointBlockSet_hasBlock(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash common.Hash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.hasBlock(tt.args.hash); got != tt.want {
				t.Errorf("hasBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_disjointBlockSet_removeBlock(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		hash common.Hash
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_disjointBlockSet_removeLowerBlocks(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	type args struct {
		num *big.Int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			fmt.Printf("s %v\n", s)
		})
	}
}

func Test_disjointBlockSet_size(t *testing.T) {
	type fields struct {
		RWMutex          sync.RWMutex
		limit            int
		blocks           map[common.Hash]*pendingBlock
		parentToChildren map[common.Hash]map[common.Hash]struct{}
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &disjointBlockSet{
				RWMutex:          tt.fields.RWMutex,
				limit:            tt.fields.limit,
				blocks:           tt.fields.blocks,
				parentToChildren: tt.fields.parentToChildren,
			}
			if got := s.size(); got != tt.want {
				t.Errorf("size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newDisjointBlockSet(t *testing.T) {
	type args struct {
		limit int
	}
	tests := []struct {
		name string
		args args
		want *disjointBlockSet
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newDisjointBlockSet(tt.args.limit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newDisjointBlockSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pendingBlock_toBlockData(t *testing.T) {
	type fields struct {
		hash          common.Hash
		number        *big.Int
		header        *types.Header
		body          *types.Body
		justification []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   *types.BlockData
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &pendingBlock{
				hash:          tt.fields.hash,
				number:        tt.fields.number,
				header:        tt.fields.header,
				body:          tt.fields.body,
				justification: tt.fields.justification,
			}
			if got := b.toBlockData(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toBlockData() = %v, want %v", got, tt.want)
			}
		})
	}
}