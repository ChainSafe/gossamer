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
	"github.com/ChainSafe/gossamer/lib/trie"

	database "github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func TestChainGetHeader_Genesis(t *testing.T) {
	state := newTestStateService(t)
	svc := NewChainModule(state.Block)

	header, err := state.Block.BestBlockHeader()
	require.NoError(t, err)

	expected := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest:         ChainBlockHeaderDigest{},
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

	expected := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest:         ChainBlockHeaderDigest{},
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

	expectedHeader := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest:         ChainBlockHeaderDigest{},
	}

	hash := state.Block.BestBlockHash()

	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *expectedHeader,
			Body:   nil,
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

	expectedHeader := &ChainBlockHeaderResponse{
		ParentHash:     header.ParentHash.String(),
		Number:         common.BytesToHex(header.Number.Bytes()),
		StateRoot:      header.StateRoot.String(),
		ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
		Digest:         ChainBlockHeaderDigest{},
	}

	expected := &ChainBlockResponse{
		Block: ChainBlock{
			Header: *expectedHeader,
			Body:   nil,
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
	expected := genesisHeader.Hash()
	require.Equal(t, common.BytesToHex(expected[:]), res)

	testhash := common.Hash{1, 2, 3, 4}
	err = state.Block.SetFinalizedHash(testhash, 77, 1)
	require.NoError(t, err)

	req = ChainFinalizedHeadRequest{77, 1}
	err = svc.GetFinalizedHeadByRound(nil, &req, &res)
	require.NoError(t, err)
	require.Equal(t, common.BytesToHex(testhash[:]), res)
}

var gen, genTrie, genesisHeader = newTestGenesisWithTrieAndHeader()

func newTestStateService(t *testing.T) *state.Service {
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)
	stateSrvc := state.NewService(testDatadirPath, log.LvlInfo)
	stateSrvc.UseMemDB()

	err = stateSrvc.Initialize(gen, genesisHeader, genTrie)
	if err != nil {
		t.Fatal(err)
	}

	err = stateSrvc.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = loadTestBlocks(genesisHeader.Hash(), stateSrvc.Block)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		stateSrvc.Stop()
	})
	return stateSrvc
}

func newTestGenesisWithTrieAndHeader() (*genesis.Genesis, *trie.Trie, *types.Header) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
	if err != nil {
		panic(err)
	}

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	if err != nil {
		panic(err)
	}

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), genTrie.MustHash(), trie.EmptyHash, types.Digest{}) //nolint
	if err != nil {
		panic(err)
	}
	return gen, genTrie, genesisHeader
}

func loadTestBlocks(gh common.Hash, bs *state.BlockState) error {
	// Create header
	header0 := &types.Header{
		Number:     big.NewInt(0),
		Digest:     types.Digest{},
		ParentHash: gh,
		StateRoot:  trie.EmptyHash,
	}
	// Create blockHash
	blockHash0 := header0.Hash()
	// BlockBody with fake extrinsics
	blockBody0 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block0 := &types.Block{
		Header: header0,
		Body:   &blockBody0,
	}

	err := bs.AddBlock(block0)
	if err != nil {
		return err
	}

	// Create header & blockData for block 1
	header1 := &types.Header{
		Number:     big.NewInt(1),
		Digest:     types.Digest{},
		ParentHash: blockHash0,
		StateRoot:  trie.EmptyHash,
	}

	// Create Block with fake extrinsics
	blockBody1 := types.Body{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	block1 := &types.Block{
		Header: header1,
		Body:   &blockBody1,
	}

	// Add the block1 to the DB
	err = bs.AddBlock(block1)
	if err != nil {
		return err
	}

	return nil
}
