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

package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

var maxRetries = 5

// testMessageTimeout is the wait time for messages to be exchanged
var testMessageTimeout = time.Second

// testGenesisHeader is a test block header
var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

// mockVerifier implements the Verifier interface
type mockVerifier struct{}

// VerifyBlock mocks verifying a block
func (v *mockVerifier) VerifyBlock(header *types.Header) (bool, error) {
	return true, nil
}

// IncrementEpoch mocks incrementing an epoch
func (v *mockVerifier) IncrementEpoch() (*babe.NextEpochDescriptor, error) {
	return &babe.NextEpochDescriptor{}, nil
}

// EpochNumber mocks an epoch number
func (v *mockVerifier) EpochNumber() uint64 {
	return 1
}

// mockBlockProducer implements the BlockProducer interface
type mockBlockProducer struct{}

// Start mocks starting
func (bp *mockBlockProducer) Start() error {
	return nil
}

// Stop mocks stopping
func (bp *mockBlockProducer) Stop() error {
	return nil
}

// Pause mocks pausing
func (bp *mockBlockProducer) Pause() error {
	return nil
}

// Resume mocks resuming
func (bp *mockBlockProducer) Resume() error {
	return nil
}

// GetBlockChannel returns a new channel
func (bp *mockBlockProducer) GetBlockChannel() <-chan types.Block {
	return make(chan types.Block)
}

// SetRuntime mocks setting runtime
func (bp *mockBlockProducer) SetRuntime(rt *runtime.Runtime) error {
	return nil
}

// mockFinalityGadget implements the FinalityGadget interface
type mockFinalityGadget struct {
	in        chan FinalityMessage
	out       chan FinalityMessage
	finalized chan FinalityMessage
}

func (fg *mockFinalityGadget) Start() error {
	return nil
}

func (fg *mockFinalityGadget) Stop() error {
	return nil
}

func (fg *mockFinalityGadget) GetVoteOutChannel() <-chan FinalityMessage {
	return fg.out
}

func (fg *mockFinalityGadget) GetVoteInChannel() chan<- FinalityMessage {
	return fg.in
}

func (fg *mockFinalityGadget) GetFinalizedChannel() <-chan FinalityMessage {
	return fg.finalized
}

func (fg *mockFinalityGadget) DecodeMessage(*network.ConsensusMessage) (FinalityMessage, error) {
	return &mockFinalityMessage{}, nil
}

var testConsensusMessage = &network.ConsensusMessage{
	ConsensusEngineID: types.GrandpaEngineID,
	Data:              []byte("nootwashere"),
}

type mockFinalityMessage struct{}

func (fm *mockFinalityMessage) ToConsensusMessage() (*network.ConsensusMessage, error) {
	return testConsensusMessage, nil
}

// NewTestService creates a new test core service
func NewTestService(t *testing.T, cfg *Config) *Service {
	if cfg == nil {
		cfg = &Config{
			IsBlockProducer: false,
		}
	}

	if cfg.Runtime == nil {
		cfg.Runtime = runtime.NewTestRuntime(t, runtime.NODE_RUNTIME)
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewKeystore()
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Keystore.Insert(kp)
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

	cfg.Verifier = &mockVerifier{}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err := stateSrvc.Initialize(genesisData, testGenesisHeader, trie.NewEmptyTrie())
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
