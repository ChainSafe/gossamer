// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSpec_ToJSON(t *testing.T) {
	tests := []struct {
		name    string
		genesis *genesis.Genesis
		want    string
		err     error
	}{
		{
			name:    "name test",
			genesis: &genesis.Genesis{Name: "test"},
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
			genesis: &genesis.Genesis{
				Name:            "test",
				ID:              "ID",
				ChainType:       "chainType",
				ProtocolID:      "protocol",
				ConsensusEngine: "babe",
			},
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
			genesis: &genesis.Genesis{
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
			},
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
				genesis: tt.genesis,
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
	file := genesis.CreateTestGenesisJSONFile(t, false)

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
	tests := []struct {
		name    string
		genesis *genesis.Genesis
		want    string
		err     error
	}{
		{
			name:    "normal conditions",
			genesis: &genesis.Genesis{Name: "test"},
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
				genesis: tt.genesis,
			}
			got, err := b.ToJSONRaw()
			assert.ErrorIs(t, err, tt.err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestWriteGenesisSpecFile(t *testing.T) {
	type args struct {
		data []byte
		fp   string
	}
	tests := []struct {
		name      string
		args      args
		err       error
		touchFile bool
	}{
		{
			name: "normal conditions",
			args: args{
				data: []byte{1},
				fp:   filepath.Join(t.TempDir(), "test.file"),
			},
		},
		{
			name: "existing file",
			args: args{
				data: []byte{1},
			},
			touchFile: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.touchFile {
				path := filepath.Join(t.TempDir(), "test.txt")
				err := os.WriteFile(path, nil, os.ModePerm)
				require.NoError(t, err)
				tt.args.fp = path
				tt.err = errors.New("file " + path + " already exists, rename to avoid overwriting")
			}
			err := WriteGenesisSpecFile(tt.args.data, tt.args.fp)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
