//go:build integration

// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	inmemory_storage "github.com/ChainSafe/gossamer/lib/runtime/storage/inmemory"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testVote2 = &Vote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 333,
	}
)

type testJustificationRequest struct {
	to  peer.ID
	num uint32
}

type testNetwork struct {
	t                    *testing.T
	out                  chan GrandpaMessage
	finalised            chan GrandpaMessage
	justificationRequest *testJustificationRequest
}

func newTestNetwork(t *testing.T) *testNetwork {
	return &testNetwork{
		t:         t,
		out:       make(chan GrandpaMessage, 128),
		finalised: make(chan GrandpaMessage, 128),
	}
}

func (n *testNetwork) GossipMessage(msg NotificationsMessage) {
	cm, ok := msg.(*ConsensusMessage)
	assert.True(n.t, ok)

	gmsg, err := decodeMessage(cm)
	assert.NoError(n.t, err)

	_, ok = gmsg.(*CommitMessage)
	if ok {
		n.finalised <- gmsg
		return
	}
	n.out <- gmsg
}

func (n *testNetwork) SendMessage(_ peer.ID, _ NotificationsMessage) error {
	return nil
}

func (n *testNetwork) SendJustificationRequest(to peer.ID, num uint32) {
	n.justificationRequest = &testJustificationRequest{
		to:  to,
		num: num,
	}
}

func (*testNetwork) RegisterNotificationsProtocol(
	_ protocol.ID,
	_ network.MessageType,
	_ network.HandshakeGetter,
	_ network.HandshakeDecoder,
	_ network.HandshakeValidator,
	_ network.MessageDecoder,
	_ network.NotificationsMessageHandler,
	_ network.NotificationsMessageBatchHandler,
	_ uint64,
) error {
	return nil
}

func (n *testNetwork) SendBlockReqestByHash(_ common.Hash) {}

func setupGrandpa(t *testing.T, kp *ed25519.Keypair) *Service {
	st := newTestState(t)
	net := newTestNetwork(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	telemetryMock.
		EXPECT().
		SendMessage(gomock.Any()).AnyTimes()

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Keypair:      kp,
		LogLvl:       log.Info,
		Authority:    true,
		Network:      net,
		Interval:     time.Second,
		Telemetry:    telemetryMock,
	}

	gs, err := NewService(cfg)
	assert.NoError(t, err)
	return gs
}

func newTestState(t *testing.T) *state.Service {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	testDatadirPath := t.TempDir()

	db, err := database.LoadDatabase(testDatadirPath, true)
	require.NoError(t, err)

	t.Cleanup(func() {
		closeErr := db.Close()
		require.NoError(t, closeErr)
	})

	_, genTrie, _ := newWestendDevGenesisWithTrieAndHeader(t)
	tries := state.NewTries()
	tries.SetTrie(genTrie)
	block, err := state.NewBlockStateFromGenesis(db, tries, testGenesisHeader, telemetryMock)
	require.NoError(t, err)

	var rtCfg wazero_runtime.Config

	rtCfg.Storage = inmemory_storage.NewTrieState(genTrie)

	rt, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)
	block.StoreRuntime(block.BestBlockHash(), rt)

	grandpa, err := state.NewGrandpaStateFromGenesis(db, nil, newTestVoters(t), telemetryMock)
	require.NoError(t, err)

	return &state.Service{
		Block:     block,
		Grandpa:   grandpa,
		Telemetry: telemetryMock,
	}
}

func newTestService(t *testing.T, keypair *ed25519.Keypair) (*Service, *state.Service) {
	st := newTestState(t)
	net := newTestNetwork(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	cfg := &Config{
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       newTestVoters(t),
		Authority:    true,
		Network:      net,
		Interval:     time.Second,
		Telemetry:    telemetryMock,
		Keypair:      keypair,
	}

	gs, err := NewService(cfg)
	require.NoError(t, err)
	return gs, st
}

func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie *trie.InMemoryTrie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	assert.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = runtime.NewInMemoryTrieFromGenesis(gen)
	assert.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash(trie.NoMaxInlineValueSize)
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeader = *types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)

	return gen, genesisTrie, genesisHeader
}
