// Copyright 2020 ChainSafe Systems (ON) Corp.
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
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
)

func TestBuildSpec_ToJSON(t *testing.T) {
	type fields struct {
		genesis *genesis.Genesis
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:    "normal conditions",
			fields:  fields{genesis: &genesis.Genesis{Name: "test"}},
			want:    []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
			wantErr: false,
		},
		// todo determine test case for error condition (How to create input to json.Marshal that will error)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuildSpec{
				genesis: tt.fields.genesis,
			}
			got, err := b.ToJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFromDB(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *BuildSpec
		wantErr bool
	}{
		{name: "invalid db path", args: args{path: "foo/bar"}, wantErr: true},
		// TODO: not sure how to test this since it's comparing pointers that I don't have reference to.
		{name: "normal conditions", args: args{path: "test_data/TestBuildFromDB"}, want: &BuildSpec{genesis: &genesis.Genesis{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFromDB(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFromDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildFromDB() got = %v, want %v", got, tt.want)
			}
			if tt.wantErr {
				// remove file created during error conditions
				err := os.RemoveAll(tt.args.path)
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildFromGenesis(t *testing.T) {
	// setup test file
	file, err := genesis.CreateTestGenesisJSONFile(false)
	require.NoError(t, err)
	defer os.Remove(file)

	type args struct {
		path      string
		authCount int
	}
	tests := []struct {
		name    string
		args    args
		want    *BuildSpec
		wantErr bool
	}{
		{
			name: "invalid file path",
			args: args{
				path:      "invalid/path",
				authCount: 0,
			},
			wantErr: true,
		},
		// TODO: not sure how to test this since it's comparing pointers that I don't have reference to.
		{
			name: "normal conditions",
			args: args{
				path:      file,
				authCount: 0,
			},
			want:    &BuildSpec{genesis: &genesis.Genesis{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFromGenesis(tt.args.path, tt.args.authCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFromGenesis() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildFromGenesis() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSpec_ToJSONRaw(t *testing.T) {
	type fields struct {
		genesis *genesis.Genesis
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:   "normal conditions",
			fields: fields{genesis: &genesis.Genesis{Name: "test"}},
			want:   []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
		},
		// todo determine test case for error condition (How to create input to json.Marshal that will error)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuildSpec{
				genesis: tt.fields.genesis,
			}
			got, err := b.ToJSONRaw()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSONRaw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToJSONRaw() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteGenesisSpecFile(t *testing.T) {

	file, err := os.Create("test.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	type args struct {
		data []byte
		fp   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "normal conditions", args: args{
			data: []byte{1},
			fp:   "test.file",
		}},
		{name: "existing file", args: args{
			data: []byte{1},
			fp:   file.Name(),
		}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteGenesisSpecFile(tt.args.data, tt.args.fp); (err != nil) != tt.wantErr {
				t.Errorf("WriteGenesisSpecFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			os.Remove(tt.args.fp)
		})
	}
}
