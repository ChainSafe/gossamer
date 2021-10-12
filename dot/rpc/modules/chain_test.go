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

package modules

import (
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"

	database "github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
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
	di.Set(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())

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
	di.Set(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())

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
	di.Set(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())

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
	di.Set(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())

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
	digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
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
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	config := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genTrie, genesisHeader := genesis.NewTestGenesisWithTrieAndHeader(t)

	err = stateSrvc.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	rt, err := stateSrvc.CreateGenesisRuntime(genTrie, gen)
	require.NoError(t, err)

	err = loadTestBlocks(t, genesisHeader.Hash(), stateSrvc.Block, rt)
	require.NoError(t, err)

	t.Cleanup(func() {
		stateSrvc.Stop()
	})
	return stateSrvc
}

func loadTestBlocks(t *testing.T, gh common.Hash, bs *state.BlockState, rt runtime.Instance) error {
	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(0),
		Digest:     types.NewDigest(),
		ParentHash: gh,
		StateRoot:  trie.EmptyHash,
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	block0 := &types.Block{
		Header: *header0,
		Body:   sampleBodyBytes,
	}

	err := bs.AddBlock(block0)
	if err != nil {
		return err
	}

	bs.StoreRuntime(block0.Header.Hash(), rt)

	// Create header & blockData for block 1
	digest := types.NewDigest()
	err = digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
	require.NoError(t, err)
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     digest,
		ParentHash: blockHash0,
		StateRoot:  trie.EmptyHash,
	}

	block1 := &types.Block{
		Header: *header1,
		Body:   sampleBodyBytes,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	if err != nil {
		return err
	}

	bs.StoreRuntime(block1.Header.Hash(), rt)

	return nil
}
