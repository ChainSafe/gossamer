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

package core

import (
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) []*types.Header {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.Nil(t, err)

	headers := []*types.Header{}

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
				Digest:     [][]byte{},
			},
			Body: &types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlock(block)
		require.Nil(t, err)
		headers = append(headers, block.Header)
	}

	return headers
}

func TestStartService(t *testing.T) {
	s := NewTestService(t, nil)

	// TODO: improve dot tests #687
	require.NotNil(t, s)

	err := s.Start()
	require.Nil(t, err)

	err = s.Stop()
	require.NoError(t, err)
}

func TestAnnounceBlock(t *testing.T) {
	net := new(mockNetwork)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		NewBlocks: newBlocks,
		Network:   net,
	}

	s := NewTestService(t, cfg)
	err := s.Start()
	require.Nil(t, err)
	defer s.Stop()

	parent := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	// simulate block sent from BABE session
	newBlocks <- types.Block{
		Header: &types.Header{
			ParentHash: parent.Hash(),
			Number:     big.NewInt(1),
		},
		Body: &types.Body{},
	}

	time.Sleep(testMessageTimeout)
	require.Equal(t, network.BlockAnnounceMsgType, net.Message.Type())
}

func TestHandleRuntimeChanges(t *testing.T) {
	tt := trie.NewEmptyTrie()
	rt := runtime.NewTestRuntimeWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	kp, err := sr25519.GenerateKeypair()
	require.Nil(t, err)

	ks := keystore.NewGlobalKeystore()
	ks.Acco.Insert(kp)

	cfg := &Config{
		Runtime:          rt,
		Keystore:         ks,
		TransactionState: state.NewTransactionState(),
		IsBlockProducer:  false,
	}

	s := NewTestService(t, cfg)

	_, err = runtime.GetRuntimeBlob(runtime.TESTS_FP, runtime.TEST_WASM_URL)
	require.Nil(t, err)

	testRuntime, err := ioutil.ReadFile(runtime.TESTS_FP)
	require.Nil(t, err)

	ts, err := s.storageState.TrieState(nil)
	require.NoError(t, err)

	err = ts.Set([]byte(":code"), testRuntime)
	require.Nil(t, err)

	root, err := ts.Root()
	require.NoError(t, err)

	s.storageState.StoreTrie(root, ts)
	head := &types.Header{
		ParentHash: s.blockState.BestBlockHash(),
		Number:     big.NewInt(1),
		StateRoot:  root,
		Digest:     [][]byte{},
	}

	err = s.blockState.AddBlock(&types.Block{
		Header: head,
		Body:   types.NewBody([]byte{}),
	})
	require.NoError(t, err)

	bestHeader, err := s.blockState.BestBlockHeader()
	require.NoError(t, err)
	require.Equal(t, head, bestHeader)

	err = s.handleRuntimeChanges(testGenesisHeader)
	require.NoError(t, err)
}

func TestService_HasKey(t *testing.T) {
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	cfg := &Config{
		Keystore: ks,
	}
	svc := NewTestService(t, cfg)

	res, err := svc.HasKey(kr.Alice().Public().Hex(), "babe")
	require.NoError(t, err)
	require.True(t, res)
}

func TestService_HasKey_UnknownType(t *testing.T) {
	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Acco.Insert(kr.Alice())

	cfg := &Config{
		Keystore: ks,
	}
	svc := NewTestService(t, cfg)

	res, err := svc.HasKey(kr.Alice().Public().Hex(), "xxxx")
	require.EqualError(t, err, "unknown key type: xxxx")
	require.False(t, res)
}
