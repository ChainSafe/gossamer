// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/scale"

	database "github.com/ChainSafe/chaindb"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/stretchr/testify/require"
)

// test data
var (
	sampleBodyBytes = *types.NewBody([]types.Extrinsic{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}})
	// sampleBodyString is string conversion of sampleBodyBytes
	sampleBodyString = []string{"0x2800010203040506070809"}
)

func TestChainGetHeader_Genesis(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	header, err := state.Block.BestBlockHeader()
	require.NoError(t, err)

	di := types.NewDigestItem()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = di.Set(*prd)
	require.NoError(t, err)

	d, err := scale.Marshal(di)
	require.NoError(t, err)

	expected := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest: ChainBlockHeaderDigest{
			Logs: []string{common.BytesToHex(d)},
		},
	}

	hash := state.Block.BestBlockHash()

	res := &ChainBlockHeaderResponse{}
	req := &ChainHashRequest{Bhash: &hash}

	err = svc.GetHeader(nil, req, res)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestChainGetHeader_Latest(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	header, err := state.Block.BestBlockHeader()
	require.NoError(t, err)

	di := types.NewDigestItem()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = di.Set(*prd)
	require.NoError(t, err)

	d, err := scale.Marshal(di)
	require.NoError(t, err)

	expected := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest: ChainBlockHeaderDigest{
			Logs: []string{common.BytesToHex(d)},
		},
	}

	res := &ChainBlockHeaderResponse{}
	req := &ChainHashRequest{} // empty request should return latest hash

	err = svc.GetHeader(nil, req, res)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestChainGetHeader_NotFound(t *testing.T) {
	chain := newTestStateService(t)
	svc := NewChainModule(chain.Block)

	bhash, err := common.HexToHash("0xea374832a2c3997280d2772c10e6e5b0b493ccd3d09c0ab14050320e34076c2c")
	require.NoError(t, err)

	res := &ChainBlockHeaderResponse{}
	req := &ChainHashRequest{Bhash: &bhash}

	err = svc.GetHeader(nil, req, res)
	require.EqualError(t, err, database.ErrKeyNotFound.Error())
}

func TestChainGetBlock_Genesis(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	header, err := state.Block.BestBlockHeader()
	require.NoError(t, err)

	di := types.NewDigestItem()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = di.Set(*prd)
	require.NoError(t, err)

	d, err := scale.Marshal(di)
	require.NoError(t, err)

	expectedHeader := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest: ChainBlockHeaderDigest{
			Logs: []string{common.BytesToHex(d)},
		},
	}

	hash := state.Block.BestBlockHash()

	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *expectedHeader,
			Body:   sampleBodyString,
		},
	}

	res := &ChainBlockResponse{}
	req := &ChainHashRequest{Bhash: &hash}

	err = svc.GetBlock(nil, req, res)
	require.Nil(t, err)

	require.Equal(t, expected, res)
}

func TestChainGetBlock_Latest(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	header, err := state.Block.BestBlockHeader()
	require.NoError(t, err)

	di := types.NewDigestItem()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = di.Set(*prd)
	require.NoError(t, err)

	d, err := scale.Marshal(di)
	require.NoError(t, err)

	expectedHeader := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest: ChainBlockHeaderDigest{
			Logs: []string{common.BytesToHex(d)},
		},
	}

	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *expectedHeader,
			Body:   sampleBodyString,
		},
	}

	res := &ChainBlockResponse{}
	req := &ChainHashRequest{} // empty request should return latest block

	err = svc.GetBlock(nil, req, res)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestChainGetBlock_NoFound(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	bhash, err := common.HexToHash("0xea374832a2c3997280d2772c10e6e5b0b493ccd3d09c0ab14050320e34076c2c")
	require.NoError(t, err)

	res := &ChainBlockResponse{}
	req := &ChainHashRequest{Bhash: &bhash}

	err = svc.GetBlock(nil, req, res)
	require.EqualError(t, err, database.ErrKeyNotFound.Error())
}

func TestChainGetBlockHash_Latest(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest{nil}

	err := svc.GetBlockHash(nil, &req, &res)
	require.Nil(t, err)

	expected := state.Block.BestBlockHash()
	require.Equal(t, expected.String(), res)
}

func TestChainGetBlockHash_ByNumber(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest{"1"}

	err := svc.GetBlockHash(nil, &req, &res)
	require.Nil(t, err)

	expected, err := state.Block.GetBlockByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, expected.Header.Hash().String(), res)
}

func TestChainGetBlockHash_ByHex(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	resString := string("")
	res := ChainHashResponse(resString)
	req := ChainBlockNumberRequest{"0x01"}

	err := svc.GetBlockHash(nil, &req, &res)
	require.NoError(t, err)

	expected, err := state.Block.GetBlockByNumber(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, expected.Header.Hash().String(), res)
}

func TestChainGetBlockHash_Array(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	resString := string("")
	res := ChainHashResponse(resString)

	nums := make([]interface{}, 2)
	nums[0] = float64(0)     // as number
	nums[1] = string("0x01") // as hex string
	req := ChainBlockNumberRequest{nums}

	err := svc.GetBlockHash(nil, &req, &res)
	require.Nil(t, err)

	expected0, err := state.Block.GetBlockByNumber(big.NewInt(0))
	require.NoError(t, err)
	expected1, err := state.Block.GetBlockByNumber(big.NewInt(1))
	require.NoError(t, err)
	expected := []string{expected0.Header.Hash().String(), expected1.Header.Hash().String()}

	require.Equal(t, expected, res)
}

func TestChainGetFinalizedHead(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)
	_, _, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	var res ChainHashResponse
	err := svc.GetFinalizedHead(nil, &EmptyRequest{}, &res)
	require.NoError(t, err)

	expected := genesisHeader.Hash()
	require.Equal(t, common.BytesToHex(expected[:]), res)
}

func TestChainGetFinalizedHeadByRound(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	var res ChainHashResponse
	req := ChainFinalizedHeadRequest{0, 0}
	err := svc.GetFinalizedHeadByRound(nil, &req, &res)
	require.NoError(t, err)

	_, _, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	expected := genesisHeader.Hash()
	require.Equal(t, common.BytesToHex(expected[:]), res)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     big.NewInt(1),
		Digest:     digest,
	}
	err = state.Block.AddBlock(&types.Block{
		Header: *header,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	testhash := header.Hash()
	err = state.Block.SetFinalisedHash(testhash, 77, 1)
	require.NoError(t, err)

	req = ChainFinalizedHeadRequest{77, 1}
	err = svc.GetFinalizedHeadByRound(nil, &req, &res)
	require.NoError(t, err)
	require.Equal(t, common.BytesToHex(testhash[:]), res)
}

func newTestStateService(t *testing.T) *state.Service {
	testDatadirPath := t.TempDir()

	config := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.Info,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	err := stateSrvc.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	rtCfg := &wasmer.Config{}

	rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
	require.NoError(t, err)

	if stateSrvc != nil {
		rtCfg.NodeStorage.BaseDB = stateSrvc.Base
	} else {
		rtCfg.NodeStorage.BaseDB, err = utils.SetupDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
		require.NoError(t, err)
	}

	rt, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	loadTestBlocks(t, genesisHeader.Hash(), stateSrvc.Block, rt)

	t.Cleanup(func() {
		stateSrvc.Stop()
	})
	return stateSrvc
}

func loadTestBlocks(t *testing.T, gh common.Hash, bs *state.BlockState, rt runtime.Instance) {
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.NewDigest(),
		ParentHash: gh,
		StateRoot:  trie.EmptyHash,
	}

	block1 := &types.Block{
		Header: *header1,
		Body:   sampleBodyBytes,
	}

	err := bs.AddBlock(block1)
	require.NoError(t, err)
	bs.StoreRuntime(header1.Hash(), rt)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	header2 := &types.Header{
		Number:     big.NewInt(2),
		Digest:     digest,
		ParentHash: header1.Hash(),
		StateRoot:  trie.EmptyHash,
	}

	block2 := &types.Block{
		Header: *header2,
		Body:   sampleBodyBytes,
	}

	err = bs.AddBlock(block2)
	require.NoError(t, err)
	bs.StoreRuntime(header2.Hash(), rt)
}
