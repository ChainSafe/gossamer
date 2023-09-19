// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateJSONRawFile(t *testing.T) {
	type args struct {
		bs *BuildSpec
		fp string
	}
	tests := []struct {
		name         string
		args         args
		expectedHash string
	}{
		{
			name: "working_example",
			args: args{
				bs: &BuildSpec{genesis: NewTestGenesis(t)},
				fp: filepath.Join(t.TempDir(), "/test.json"),
			},
			expectedHash: "b94270ba29cd934f8d6ceb4d94929884d46b356ff7a8135b85e2eed9e0216abf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CreateJSONRawFile(tt.args.bs, tt.args.fp)

			b, err := os.ReadFile(tt.args.fp)
			require.NoError(t, err)
			digest := sha256.Sum256(b)
			hexDigest := fmt.Sprintf("%x", digest)
			require.Equal(t, tt.expectedHash, hexDigest)
		})
	}
}

func TestNewTestGenesisFile(t *testing.T) {
	type args struct {
		t      *testing.T
		config *cfg.Config
	}
	tests := []struct {
		name string
		args args
		want *os.File
	}{
		{
			name: "working_example",
			args: args{
				t:      t,
				config: DefaultTestWestendDevConfig(t),
			},
			want: &os.File{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTestGenesisRawFile(tt.args.t, tt.args.config)
			if tt.want != nil {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestRandomNodeName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "working_example",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandomNodeName()
			assert.Regexp(t, "[a-z]*-[a-z]*-[0-9]*", got)
		})
	}
}

// NewTestGenesis returns a test genesis instance using "westend-dev" raw data
func NewTestGenesis(t *testing.T) *genesis.Genesis {
	fp := utils.GetWestendDevRawGenesisPath(t)

	westendDevGenesis, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	return &genesis.Genesis{
		Name:       "test",
		ID:         "test",
		Bootnodes:  []string(nil),
		ProtocolID: "/gossamer/test/0",
		Genesis:    westendDevGenesis.GenesisFields(),
	}
}
