// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStateFile(t *testing.T) string {
	t.Helper()

	const filename = "../lib/runtime/test_data/kusama/block1482002.out"

	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	require.NoError(t, err)
	pairs := rpcPairs["result"].([]interface{})

	bz, err := json.Marshal(pairs)
	require.NoError(t, err)

	fp := filepath.Join(t.TempDir(), "state.json")
	err = ioutil.WriteFile(fp, bz, 0777)
	require.NoError(t, err)

	return fp
}

func TestImportState(t *testing.T) {
	t.Parallel()

	// setup node for test
	basepath, err := ioutil.TempDir("", "gossamer-test-*")
	require.NoError(t, err)

	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := newTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	cfg.Init.Genesis = genFile

	cfg.Global.BasePath = basepath
	nodeInstance := nodeBuilder{}
	err = nodeInstance.initNode(cfg)
	require.NoError(t, err)

	stateFP := setupStateFile(t)
	headerFP := setupHeaderFile(t)

	type args struct {
		basepath  string
		stateFP   string
		headerFP  string
		firstSlot uint64
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "no arguments",
			err:  errors.New("read .: is a directory"),
		},
		{
			name: "working example",
			args: args{
				basepath:  basepath,
				stateFP:   stateFP,
				headerFP:  headerFP,
				firstSlot: 262493679,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ImportState(tt.args.basepath, tt.args.stateFP, tt.args.headerFP, tt.args.firstSlot)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_newHeaderFromFile(t *testing.T) {
	t.Parallel()

	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want *types.Header
		err  error
	}{
		{
			name: "working example",
			args: args{filename: setupHeaderFile(t)},
			want: &types.Header{
				ParentHash:     common.MustHexToHash("0x3b45c9c22dcece75a30acc9c2968cb311e6b0557350f83b430f47559db786975"),
				Number:         big.NewInt(1482002),
				StateRoot:      common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
				ExtrinsicsRoot: common.MustHexToHash("0xda26dc8c1455f8f81cae12e4fc59e23ce961b2c837f6d3f664283af906d344e0"),
			},
		},
		{
			name: "no arguments",
			err:  errors.New("read .: is a directory"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newHeaderFromFile(tt.args.filename)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want != nil {
				assert.Equal(t, tt.want.ParentHash, got.ParentHash)
				assert.Equal(t, tt.want.Number, got.Number)
				assert.Equal(t, tt.want.StateRoot, got.StateRoot)
				assert.Equal(t, tt.want.ExtrinsicsRoot, got.ExtrinsicsRoot)
			}
		})
	}
}

func Test_newTrieFromPairs(t *testing.T) {
	t.Parallel()

	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want common.Hash
		err  error
	}{
		{
			name: "no arguments",
			err:  errors.New("read .: is a directory"),
			want: common.Hash{},
		},
		{
			name: "working example",
			args: args{filename: setupStateFile(t)},
			want: common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTrieFromPairs(tt.args.filename)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}

			if !tt.want.IsEmpty() {
				assert.Equal(t, tt.want, got.MustHash())
			}
		})
	}
}
