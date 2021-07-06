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

package sync

import (
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"

	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
)

// NewMockFinalityGadget create and return sync FinalityGadget interface mock
func NewMockFinalityGadget() *syncmocks.MockFinalityGadget {
	m := new(syncmocks.MockFinalityGadget)
	// using []uint8 instead of []byte: https://github.com/stretchr/testify/pull/969
	m.On("VerifyBlockJustification", mock.AnythingOfType("[]uint8")).Return(nil)
	return m
}

// NewMockVerifier create and return sync Verifier interface mock
func NewMockVerifier() *syncmocks.MockVerifier {
	m := new(syncmocks.MockVerifier)
	m.On("VerifyBlock", mock.AnythingOfType("*types.Header")).Return(nil)
	return m
}

// NewTestSyncer ...
func NewTestSyncer(t *testing.T, usePolkadotGenesis bool) *Service {
	wasmer.DefaultTestLogLvl = 3

	cfg := &Config{}
	testDatadirPath, _ := ioutil.TempDir("/tmp", "test-datadir-*")

	scfg := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t, usePolkadotGenesis)
	err := stateSrvc.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	cfg.BlockImportHandler = new(syncmocks.MockBlockImportHandler)
	cfg.BlockImportHandler.(*syncmocks.MockBlockImportHandler).On("HandleBlockImport", mock.AnythingOfType("*types.Block"), mock.AnythingOfType("*storage.TrieState")).Return(nil)

	if cfg.Runtime == nil {
		// set state to genesis state
		genState, err := rtstorage.NewTrieState(genTrie) //nolint
		require.NoError(t, err)

		rtCfg := &wasmer.Config{}
		rtCfg.Storage = genState
		rtCfg.LogLvl = 3

		instance, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg) //nolint
		require.NoError(t, err)
		cfg.Runtime = instance
	}

	if cfg.TransactionState == nil {
		cfg.TransactionState = stateSrvc.Transaction
	}

	if cfg.Verifier == nil {
		cfg.Verifier = NewMockVerifier()
	}

	if cfg.LogLvl == 0 {
		cfg.LogLvl = log.LvlInfo
	}

	if cfg.FinalityGadget == nil {
		cfg.FinalityGadget = NewMockFinalityGadget()
	}

	syncer, err := NewService(cfg)
	require.NoError(t, err)
	return syncer
}

func newTestGenesisWithTrieAndHeader(t *testing.T, usePolkadotGenesis bool) (*genesis.Genesis, *trie.Trie, *types.Header) {
	fp := "../../chain/gssmr/genesis.json"
	if usePolkadotGenesis {
		fp = "../../chain/polkadot/genesis.json"
	}

	gen, err := genesis.NewGenesisFromJSONRaw(fp)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

// BuildBlock ...
func BuildBlock(t *testing.T, instance runtime.Instance, parent *types.Header, ext types.Extrinsic) *types.Block {
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     big.NewInt(0).Add(parent.Number, big.NewInt(1)),
		Digest:     types.Digest{},
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	var exts [][]byte
	err = scale.Unmarshal(inherentExts, &exts)
	require.NoError(t, err)

	inExt := exts

	var body *types.Body
	if ext != nil {
		var txn *transaction.Validity
		externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
		txn, err = instance.ValidateTransaction(externalExt)
		require.NoError(t, err)

		vtx := transaction.NewValidTransaction(ext, txn)
		_, err = instance.ApplyExtrinsic(ext) // TODO: Determine error for ret
		require.NoError(t, err)

		body, err = babe.ExtrinsicsToBody(inExt, []*transaction.ValidTransaction{vtx})
		require.NoError(t, err)

	} else {
		body = types.NewBody(inherentExts)
	}

	// apply each inherent extrinsic
	for _, ext := range inExt {
		in, err := scale.Marshal(ext) //nolint
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)
	res.Number = header.Number

	return &types.Block{
		Header: res,
		Body:   body,
	}
}
