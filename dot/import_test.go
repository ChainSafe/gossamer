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

package dot

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/trie"
)

func TestImportState(t *testing.T) {
	type args struct {
		basepath  string
		stateFP   string
		headerFP  string
		firstSlot uint64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ImportState(tt.args.basepath, tt.args.stateFP, tt.args.headerFP, tt.args.firstSlot); (err != nil) != tt.wantErr {
				t.Errorf("ImportState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newHeaderFromFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Header
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newHeaderFromFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("newHeaderFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newHeaderFromFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newTrieFromPairs(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    *trie.Trie
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTrieFromPairs(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTrieFromPairs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTrieFromPairs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
