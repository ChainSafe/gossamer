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
	"io"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

// testMessageTimeout is the wait time for messages to be exchanged
var testMessageTimeout = time.Second

func newTestGenesisWithTrieAndHeader(t *testing.T) (*genesis.Genesis, *trie.Trie, *types.Header) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../chain/gssmr/genesis.json")
	if err != nil {
		gen, err = genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
		require.NoError(t, err)
	}

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), genTrie.MustHash(), trie.EmptyHash, types.Digest{})
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}

type mockVerifier struct{}

func (v *mockVerifier) SetOnDisabled(_ uint32, _ *types.Header) error {
	return nil
}

// mockBlockProducer implements the BlockProducer interface
type mockBlockProducer struct {
	disabled uint32
}

// Start mocks starting
func (bp *mockBlockProducer) Start() error {
	return nil
}

// Stop mocks stopping
func (bp *mockBlockProducer) Stop() error {
	return nil
}

func (bp *mockBlockProducer) SetOnDisabled(idx uint32) {
	bp.disabled = idx
}

// GetBlockChannel returns a new channel
func (bp *mockBlockProducer) GetBlockChannel() <-chan types.Block {
	return make(chan types.Block)
}

// SetRuntime mocks setting runtime
func (bp *mockBlockProducer) SetRuntime(rt runtime.Instance) {}

type mockNetwork struct {
	Message network.Message
}

func (n *mockNetwork) SendMessage(m network.NotificationsMessage) {
	n.Message = m
}

// mockFinalityGadget implements the FinalityGadget interface
type mockFinalityGadget struct {
	auths []*types.Authority
}

// Start mocks starting
func (fg *mockFinalityGadget) Start() error {
	return nil
}

// Stop mocks stopping
func (fg *mockFinalityGadget) Stop() error {
	return nil
}

func (fg *mockFinalityGadget) UpdateAuthorities(ad []*types.Authority) {
	fg.auths = ad
}

func (fg *mockFinalityGadget) Authorities() []*types.Authority {
	return fg.auths
}

// NewTestService creates a new test core service
func NewTestService(t *testing.T, cfg *Config) *Service {
	if cfg == nil {
		cfg = &Config{
			IsBlockProducer: false,
		}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewGlobalKeystore()
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		cfg.Keystore.Acco.Insert(kp)
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.Verifier == nil {
		cfg.Verifier = new(mockVerifier)
	}

	cfg.LogLvl = 3

	var stateSrvc *state.Service
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)

	if cfg.BlockState == nil || cfg.StorageState == nil || cfg.TransactionState == nil || cfg.EpochState == nil {
		stateSrvc = state.NewService(testDatadirPath, log.LvlInfo)
		stateSrvc.UseMemDB()

		err = stateSrvc.Initialize(gen, genHeader, genTrie)
		require.Nil(t, err)

		err = stateSrvc.Start()
		require.Nil(t, err)
	}

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	if cfg.TransactionState == nil {
		cfg.TransactionState = stateSrvc.Transaction
	}

	if cfg.EpochState == nil {
		cfg.EpochState = stateSrvc.Epoch
	}

	if cfg.Runtime == nil {
		rtCfg := &wasmer.Config{}
		rtCfg.Storage, err = rtstorage.NewTrieState(genTrie)
		require.NoError(t, err)
		cfg.Runtime, err = wasmer.NewRuntimeFromGenesis(gen, rtCfg)
		require.NoError(t, err)
	}

	if cfg.Network == nil {
		config := &network.Config{
			BasePath:           testDatadirPath,
			Port:               7001,
			RandSeed:           1,
			NoBootstrap:        true,
			NoMDNS:             true,
			BlockState:         stateSrvc.Block,
			TransactionHandler: &mockTransactionHandler{},
		}
		cfg.Network = createTestNetworkService(t, config)
	}

	s, err := NewService(cfg)
	require.Nil(t, err)

	if net, ok := cfg.Network.(*network.Service); ok {
		net.SetTransactionHandler(s)
		_ = net.Stop()
	}

	return s
}

// helper method to create and start a new network service
func createTestNetworkService(t *testing.T, cfg *network.Config) (srvc *network.Service) {
	if cfg.LogLvl == 0 {
		cfg.LogLvl = 3
	}

	if cfg.Syncer == nil {
		cfg.Syncer = newMockSyncer()
	}

	srvc, err := network.NewService(cfg)
	require.NoError(t, err)

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := srvc.Stop()
		require.NoError(t, err)
	})
	return srvc
}

type mockSyncer struct {
	highestSeen *big.Int
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{
		highestSeen: big.NewInt(0),
	}
}

func (s *mockSyncer) CreateBlockResponse(msg *network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	return nil, nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *network.BlockAnnounceMessage) error {
	return nil
}

func (s *mockSyncer) ProcessBlockData(_ []*types.BlockData) (int, error) {
	return 0, nil
}

func (s *mockSyncer) IsSynced() bool {
	return false
}

func (s *mockSyncer) SetSyncing(bool) {}

type mockDigestItem struct { //nolint
	i int
}

func newMockDigestItem(i int) *mockDigestItem { //nolint
	return &mockDigestItem{
		i: i,
	}
}

func (d *mockDigestItem) String() string { //nolint
	return ""
}

func (d *mockDigestItem) Type() byte { //nolint
	return byte(d.i)
}

func (d *mockDigestItem) Encode() ([]byte, error) { //nolint
	return []byte{byte(d.i)}, nil
}

func (d *mockDigestItem) Decode(_ io.Reader) error { //nolint
	return nil
}

type mockTransactionHandler struct{}

func (h *mockTransactionHandler) HandleTransactionMessage(_ *network.TransactionMessage) error {
	return nil
}
