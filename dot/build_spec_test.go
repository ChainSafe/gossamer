// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
)

func TestBuildSpec_ToJSON(t *testing.T) {
	type fields struct {
		genesis *genesis.Genesis
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
		err    error
	}{
		{
			name:   "name test",
			fields: fields{genesis: &genesis.Genesis{Name: "test"}},
			want:   []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
		},
		{
			name: "additional parameters test",
			fields: fields{genesis: &genesis.Genesis{
				Name:            "test",
				ID:              "ID",
				ChainType:       "chainType",
				ProtocolID:      "protocol",
				ConsensusEngine: "babe",
			}},
			want: []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 73, 68, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
		},
		{
			name: "normal conditions",
			fields: fields{genesis: &genesis.Genesis{
				Name:               "test",
				ID:                 "ID",
				ChainType:          "chainType",
				Bootnodes:          []string{"node1", "node2"},
				TelemetryEndpoints: []interface{}{"endpoint"},
				ProtocolID:         "protocol",
				Genesis:            genesis.Fields{},
				Properties:         map[string]interface{}{"key": "value"},
				ForkBlocks:         []string{"1", "2"},
				BadBlocks:          []string{"3", "4"},
				ConsensusEngine:    "babe",
				CodeSubstitutes:    map[string]string{"key": "value"},
			}},
			want: []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 73, 68, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 91, 10, 32, 32, 32, 32, 32, 32, 32, 32, 34, 110, 111, 100, 101, 49, 34, 44, 10, 32, 32, 32, 32, 32, 32, 32, 32, 34, 110, 111, 100, 101, 50, 34, 10, 32, 32, 32, 32, 93, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 123, 10, 32, 32, 32, 32, 32, 32, 32, 32, 34, 107, 101, 121, 34, 58, 32, 34, 118, 97, 108, 117, 101, 34, 10, 32, 32, 32, 32, 125, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuildSpec{
				genesis: tt.fields.genesis,
			}
			got, err := b.ToJSON()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFromDB(t *testing.T) {
	// initialise node (initialise state database and load genesis data)
	cfg := NewTestConfig(t)
	cfg.Init.Genesis = "../chain/gssmr/genesis.json"
	err := InitNode(cfg)
	assert.NoError(t, err)

	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want *BuildSpec
		err  error
	}{
		{name: "normal conditions", args: args{path: "test_data/TestBuildFromDB"}, want: &BuildSpec{genesis: &genesis.Genesis{Name: "Gossamer"}}},
		{name: "invalid db path", args: args{path: "foo/bar"}, err: errors.New("failed to create block state: cannot get block 0: Key not found")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFromDB(tt.args.path)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.want != nil {
				assert.Equal(t, tt.want.genesis.Name, got.genesis.Name)
			}
			// remove files created for tests
			err = os.RemoveAll(tt.args.path)
			assert.NoError(t, err)
		})
	}
}

func TestBuildFromGenesis(t *testing.T) {
	// setup test file
	file, err := genesis.CreateTestGenesisJSONFile(false)
	assert.NoError(t, err)
	defer os.Remove(file)

	type args struct {
		path      string
		authCount int
	}
	tests := []struct {
		name string
		args args
		want *BuildSpec
		err  error
	}{
		{
			name: "invalid file path",
			args: args{
				path: "/invalid/path",
			},
			err: errors.New("open /invalid/path: no such file or directory"),
		},
		{
			name: "normal conditions",
			args: args{
				path: file,
			},
			want: &BuildSpec{genesis: &genesis.Genesis{Name: "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFromGenesis(tt.args.path, tt.args.authCount)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.want != nil {
				assert.Equal(t, tt.want.genesis.Name, got.genesis.Name)
			}
		})
	}
}

func TestBuildSpec_ToJSONRaw(t *testing.T) {
	type fields struct {
		genesis *genesis.Genesis
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
		err    error
	}{
		{
			name:   "normal conditions",
			fields: fields{genesis: &genesis.Genesis{Name: "test"}},
			want:   []byte{123, 10, 32, 32, 32, 32, 34, 110, 97, 109, 101, 34, 58, 32, 34, 116, 101, 115, 116, 34, 44, 10, 32, 32, 32, 32, 34, 105, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 104, 97, 105, 110, 84, 121, 112, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 98, 111, 111, 116, 78, 111, 100, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 116, 101, 108, 101, 109, 101, 116, 114, 121, 69, 110, 100, 112, 111, 105, 110, 116, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 116, 111, 99, 111, 108, 73, 100, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 103, 101, 110, 101, 115, 105, 115, 34, 58, 32, 123, 125, 44, 10, 32, 32, 32, 32, 34, 112, 114, 111, 112, 101, 114, 116, 105, 101, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 102, 111, 114, 107, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 98, 97, 100, 66, 108, 111, 99, 107, 115, 34, 58, 32, 110, 117, 108, 108, 44, 10, 32, 32, 32, 32, 34, 99, 111, 110, 115, 101, 110, 115, 117, 115, 69, 110, 103, 105, 110, 101, 34, 58, 32, 34, 34, 44, 10, 32, 32, 32, 32, 34, 99, 111, 100, 101, 83, 117, 98, 115, 116, 105, 116, 117, 116, 101, 115, 34, 58, 32, 110, 117, 108, 108, 10, 125},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuildSpec{
				genesis: tt.fields.genesis,
			}
			got, err := b.ToJSONRaw()
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteGenesisSpecFile(t *testing.T) {
	file, err := os.Create("test.txt")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	type args struct {
		data []byte
		fp   string
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{name: "normal conditions", args: args{
			data: []byte{1},
			fp:   "test.file",
		}},
		{name: "existing file", args: args{
			data: []byte{1},
			fp:   file.Name(),
		}, err: errors.New("file test.txt already exists, rename to avoid overwriting")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteGenesisSpecFile(tt.args.data, tt.args.fp)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			os.Remove(tt.args.fp)
		})
	}
}
