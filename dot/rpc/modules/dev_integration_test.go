// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"encoding/binary"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var genesisBABEConfig = &types.BabeConfiguration{
	SlotDuration:       1000,
	EpochLength:        200,
	C1:                 1,
	C2:                 4,
	GenesisAuthorities: []types.AuthorityRaw{},
	Randomness:         [32]byte{},
	SecondarySlots:     0,
}

func newState(t *testing.T) (*state.BlockState, *state.EpochState) {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	db := state.NewInMemoryDB(t)

	_, genesisTrie, genesisHeader := newWestendLocalGenesisWithTrieAndHeader(t)
	tries := state.NewTries()
	tries.SetTrie(genesisTrie)
	bs, err := state.NewBlockStateFromGenesis(db, tries, &genesisHeader, telemetryMock)
	require.NoError(t, err)
	es, err := state.NewEpochStateFromGenesis(db, bs, genesisBABEConfig)
	require.NoError(t, err)
	return bs, es
}

func newBABEService(t *testing.T) *babe.Service {
	ctrl := gomock.NewController(t)

	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	bs, es := newState(t)
	tt := trie.NewEmptyTrie()
	rt := wazero_runtime.NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929, wazero_runtime.TestWithTrie(tt))
	bs.StoreRuntime(bs.GenesisHash(), rt)
	tt.Put(
		common.MustHexToBytes("0x886726f904d8372fdabb7707870c2fad"),
		common.MustHexToBytes("0x24d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000"+
			"0000008eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48010000000000000090b5ab205c697"+
			"4c9ea841be688864633dc9ca8a357843eeacf2314649965fe220100000000000000306721211d5404bd9da88e0204360a1a"+
			"9ab8b87c66c1bc2fcdd37f3c2222cc200100000000000000e659a7a1628cdd93febc04a4e0646ea20e9f5f0ce097d9a0529"+
			"0d4a9e054df4e01000000000000001cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c010000"+
			"00000000004603307f855321776922daeea21ee31720388d097cdaac66f05a6f8462b317570100000000000000be1d9d59d"+
			"e1283380100550a7b024501cb62d6cc40e3db35fcc5cf341814986e01000000000000001206960f920a23f7f4c43cc9081"+
			"ec2ed0721f31a9bef2c10fd7602e16e08a32c0100000000000000"))

	cfg := &babe.ServiceConfig{
		BlockState:         bs,
		EpochState:         es,
		Keypair:            kr.Alice().(*sr25519.Keypair),
		IsDev:              true,
		BlockImportHandler: NewMockBlockImportHandler(ctrl),
	}

	babe, err := babe.NewService(cfg)
	require.NoError(t, err)
	err = babe.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = babe.Stop()
	})
	return babe
}

func TestDevControl_Babe(t *testing.T) {
	t.Skip() // skip for now, blocks on `babe.Service.Resume()`
	bs := newBABEService(t)
	m := NewDevModule(bs, nil)

	var res string
	err := m.Control(nil, &[]string{"babe", "stop"}, &res)
	require.NoError(t, err)
	require.Equal(t, blockProducerStoppedMsg, res)
	require.True(t, bs.IsPaused())

	err = m.Control(nil, &[]string{"babe", "start"}, &res)
	require.NoError(t, err)
	require.Equal(t, blockProducerStartedMsg, res)
	require.False(t, bs.IsPaused())
}

func TestDevControl_Network(t *testing.T) {
	net := newNetworkService(t)
	m := NewDevModule(nil, net)

	var res string
	err := m.Control(nil, &[]string{"network", "stop"}, &res)
	require.NoError(t, err)
	require.Equal(t, networkStoppedMsg, res)
	require.True(t, net.IsStopped())

	err = m.Control(nil, &[]string{"network", "start"}, &res)
	require.NoError(t, err)
	require.Equal(t, networkStartedMsg, res)
	require.False(t, net.IsStopped())
}

func TestDevControl_SlotDuration(t *testing.T) {
	bs := newBABEService(t)
	m := NewDevModule(bs, nil)

	slotDurationSource := m.blockProducerAPI.SlotDuration()

	var res string
	err := m.SlotDuration(nil, &EmptyRequest{}, &res)
	require.NoError(t, err)

	slotLengthFetched := binary.LittleEndian.Uint64(common.MustHexToBytes(res))
	require.Equal(t, slotDurationSource, slotLengthFetched)
}

func TestDevControl_EpochLength(t *testing.T) {
	bs := newBABEService(t)
	m := NewDevModule(bs, nil)

	epochLengthSource := m.blockProducerAPI.EpochLength()

	var res string
	err := m.EpochLength(nil, &EmptyRequest{}, &res)
	require.NoError(t, err)

	epochLengthFetched := binary.LittleEndian.Uint64(common.MustHexToBytes(res))
	require.Equal(t, epochLengthSource, epochLengthFetched)
}
