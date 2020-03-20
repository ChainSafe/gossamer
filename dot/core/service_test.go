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
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/tests"
	"github.com/stretchr/testify/require"
)

// TestMessageTimeout is the wait time for messages to be exchanged
var TestMessageTimeout = time.Second

// TestHeader is a test block header
var TestHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

// newTestService creates a new test core service
func newTestService(t *testing.T, cfg *Config) *Service {
	if cfg == nil {
		rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)
		cfg = &Config{
			Runtime:     rt,
			IsAuthority: false,
		}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewKeystore()
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.MsgRec == nil {
		cfg.MsgRec = make(chan network.Message, 10)
	}

	if cfg.MsgSend == nil {
		cfg.MsgSend = make(chan network.Message, 10)
	}

	if cfg.SyncChan == nil {
		cfg.SyncChan = make(chan *big.Int, 10)
	}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	err := stateSrvc.Initialize(TestHeader, trie.NewEmptyTrie(nil))
	require.Nil(t, err)

	err = stateSrvc.Start()
	require.Nil(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	s, err := NewService(cfg)
	require.Nil(t, err)

	return s
}

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.Nil(t, err)

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
			},
			Body: &types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlock(block)
		require.Nil(t, err)
	}
}

func TestStartService(t *testing.T) {
	s := newTestService(t, nil)
	require.NotNil(t, s) // TODO: improve dot core tests

	err := s.Start()
	require.Nil(t, err)

	s.Stop()
}

func TestNotAuthority(t *testing.T) {
	cfg := &Config{
		Keystore:    keystore.NewKeystore(),
		IsAuthority: false,
	}

	s := newTestService(t, cfg)
	if s.bs != nil {
		t.Fatal("Fail: should not have babe session")
	}
}

func TestAnnounceBlock(t *testing.T) {
	msgSend := make(chan network.Message)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		NewBlocks: newBlocks,
		MsgSend:   msgSend,
	}

	s := newTestService(t, cfg)
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

	select {
	case msg := <-msgSend:
		msgType := msg.GetType()
		if !reflect.DeepEqual(msgType, network.BlockAnnounceMsgType) {
			t.Error(
				"received unexpected message type",
				"\nexpected:", network.BlockAnnounceMsgType,
				"\nreceived:", msgType,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}
