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
		want   string
		err    error
	}{
		{
			name:   "name test",
			fields: fields{genesis: &genesis.Genesis{Name: "test"}},
			want: `{
    "name": "test",
    "id": "",
    "chainType": "",
    "bootNodes": null,
    "telemetryEndpoints": null,
    "protocolId": "",
    "genesis": {},
    "properties": null,
    "forkBlocks": null,
    "badBlocks": null,
    "consensusEngine": "",
    "codeSubstitutes": null
}`,
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
			want: `{
    "name": "test",
    "id": "ID",
    "chainType": "chainType",
    "bootNodes": null,
    "telemetryEndpoints": null,
    "protocolId": "protocol",
    "genesis": {},
    "properties": null,
    "forkBlocks": null,
    "badBlocks": null,
    "consensusEngine": "",
    "codeSubstitutes": null
}`,
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
			want: `{
    "name": "test",
    "id": "ID",
    "chainType": "chainType",
    "bootNodes": [
        "node1",
        "node2"
    ],
    "telemetryEndpoints": null,
    "protocolId": "protocol",
    "genesis": {},
    "properties": {
        "key": "value"
    },
    "forkBlocks": null,
    "badBlocks": null,
    "consensusEngine": "",
    "codeSubstitutes": null
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BuildSpec{
				genesis: tt.fields.genesis,
			}
			got, err := b.ToJSON()
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestBuildFromDB(t *testing.T) {
	// initialise node (initialise state database and load genesis data)
	cfg := NewTestConfig(t)
	cfg.Init.Genesis = "../chain/gssmr/genesis.json"
	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(cfg)
	assert.NoError(t, err)

	basePath := t.TempDir()
	expectedPath := basePath[:len(basePath)-1] + "1"

	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want *BuildSpec
		err  error
	}{
		{name: "normal conditions", args: args{path: expectedPath},
			want: &BuildSpec{genesis: &genesis.Genesis{Name: "Gossamer"}}},
		{name: "invalid db path", args: args{path: "foo/bar"},
			err: errors.New("cannot start state service: failed to create block state: cannot get block 0: Key not found")},
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
	file := genesis.CreateTestGenesisJSONFile(t, false)
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
		want   string
		err    error
	}{
		{
			name:   "normal conditions",
			fields: fields{genesis: &genesis.Genesis{Name: "test"}},
			want: `{
    "name": "test",
    "id": "",
    "chainType": "",
    "bootNodes": null,
    "telemetryEndpoints": null,
    "protocolId": "",
    "genesis": {},
    "properties": null,
    "forkBlocks": null,
    "badBlocks": null,
    "consensusEngine": "",
    "codeSubstitutes": null
}`,
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
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestWriteGenesisSpecFile(t *testing.T) {
	file, err := os.CreateTemp("", "test.txt")
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
		}, err: errors.New("file " + file.Name() + " already exists, rename to avoid overwriting")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteGenesisSpecFile(tt.args.data, tt.args.fp)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.args.fp != "" {
				os.Remove(tt.args.fp)
			}
		})
	}
}
