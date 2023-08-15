// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package dot

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newTrieFromPairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		version  trie.Version
		want     common.Hash
		err      error
	}{
		{
			name: "no_arguments",
			err:  errors.New("read .: is a directory"),
			want: common.Hash{},
		},
		{
			name:     "working example",
			filename: setupStateFile(t),
			version:  trie.V0,
			want:     common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newTrieFromPairs(tt.filename, tt.version)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.want.IsEmpty() {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, got.MustHash())
			}
		})
	}
}

func TestNewHeaderFromFile(t *testing.T) {
	fp := setupHeaderFile(t)
	header, err := newHeaderFromFile(fp)
	require.NoError(t, err)

	digestBytes := common.MustHexToBytes("0x080642414245b501013c0000009659bd0f0000000070edad1c9064fff78cb18435223d8adaf5ea04c24b1a8766e3dc01eb03cc6a0c11b79793d4e31cc0990838229c44fed1669a7c7c79e1e6d0a96374d6496728069d1ef739e290497a0e3b728fa88fcbdd3a5504e0efde0242e7a806dd4fa9260c054241424501019e7f28dddcf27c1e6b328d5694c368d5b2ec5dbe0e412ae1c98f88d53be4d8502fac571f3f19c9caaf281a673319241e0c5095a683ad34316204088a36a4bd86") //nolint:lll
	digest := types.NewDigest()
	err = scale.Unmarshal(digestBytes, &digest)
	require.NoError(t, err)
	require.Len(t, digest.Types, 2)

	expected := &types.Header{
		ParentHash:     common.MustHexToHash("0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975"),
		Number:         1482002,
		StateRoot:      common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		ExtrinsicsRoot: common.MustHexToHash("0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0"),
		Digest:         digest,
	}

	require.Equal(t, expected, header)
}

func TestImportState_Integration(t *testing.T) {
	config := DefaultTestWestendDevConfig(t)

	genFile := NewTestGenesisRawFile(t, config)

	config.ChainSpec = genFile
	err := InitNode(config)
	require.NoError(t, err)

	stateFP := setupStateFile(t)
	headerFP := setupHeaderFile(t)

	const firstSlot = uint64(262493679)
	err = ImportState(config.BasePath, stateFP, headerFP, firstSlot, trie.V0)
	require.NoError(t, err)
	// confirm data is imported into db
	stateConfig := state.Config{
		Path:     config.BasePath,
		LogLevel: log.Info,
	}
	srv := state.NewService(stateConfig)
	srv.SetupBase()

	lookupKey := []byte{98, 108, 111, 99, 107, 104, 100, 114, 88, 127, 109, 161, 191, 167, 26, 103, 95, 16, 223, 160,
		246, 62, 223, 207, 22, 142, 142, 206, 151, 235, 95, 82, 106, 175, 14, 138, 142, 130, 219, 63}
	data, err := srv.DB().Get(lookupKey)
	require.NoError(t, err)
	require.NotNil(t, data)
}

func TestImportState(t *testing.T) {
	t.Parallel()

	config := DefaultTestWestendDevConfig(t)

	config.ChainSpec = NewTestGenesisRawFile(t, config)
	nodeInstance := nodeBuilder{}
	err := nodeInstance.initNode(config)
	require.NoError(t, err)

	stateFP := setupStateFile(t)
	headerFP := setupHeaderFile(t)

	type args struct {
		basepath  string
		stateFP   string
		version   trie.Version
		headerFP  string
		firstSlot uint64
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "no_arguments",
			err:  errors.New("read .: is a directory"),
		},
		{
			name: "working_example",
			args: args{
				basepath:  config.BasePath,
				stateFP:   stateFP,
				headerFP:  headerFP,
				firstSlot: 262493679,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ImportState(tt.args.basepath, tt.args.stateFP, tt.args.headerFP, tt.args.firstSlot, tt.args.version)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
