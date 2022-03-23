// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"testing"
	"time"

	"context"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

//go:generate mockgen -destination=mock_telemetry_test.go -package $GOPACKAGE github.com/ChainSafe/gossamer/dot/telemetry Client

func newTestHandler(t *testing.T) (*Handler, *state.Service) {
	testDatadirPath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		Telemetry: telemetryMock,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := genesis.NewTestGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	dh, err := NewHandler(log.Critical, stateSrvc.Block, stateSrvc.Epoch, stateSrvc.Grandpa)
	require.NoError(t, err)
	return dh, stateSrvc
}

func TestHandler_GrandpaScheduledChange(t *testing.T) {
	handler, _ := newTestHandler(t)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	sc := types.GrandpaScheduledChange{
		Auths: []types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: 3,
	}

	var digest = types.NewGrandpaConsensusDigest()
	err = digest.Set(sc)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: 1,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	headers, _ := state.AddBlocksToState(t, handler.blockState.(*state.BlockState), 2, false)
	for i, h := range headers {
		err = handler.blockState.(*state.BlockState).SetFinalisedHash(h.Hash(), uint64(i), 0)
		require.NoError(t, err)
	}

	// authorities should change on start of block 3 from start
	headers, _ = state.AddBlocksToState(t, handler.blockState.(*state.BlockState), 1, false)
	for _, h := range headers {
		err = handler.blockState.(*state.BlockState).SetFinalisedHash(h.Hash(), 3, 0)
		require.NoError(t, err)
	}

	time.Sleep(time.Millisecond * 500)
	setID, err := handler.grandpaState.(*state.GrandpaState).GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(setID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestHandler_GrandpaForcedChange(t *testing.T) {
	handler, _ := newTestHandler(t)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	fc := types.GrandpaForcedChange{
		Auths: []types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: 3,
	}

	var digest = types.NewGrandpaConsensusDigest()
	err = digest.Set(fc)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: 1,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	// authorities should change on start of block 4 from start
	state.AddBlocksToState(t, handler.blockState.(*state.BlockState), 4, false)
	time.Sleep(time.Millisecond * 100)

	setID, err := handler.grandpaState.(*state.GrandpaState).GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(setID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(fc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestHandler_GrandpaPauseAndResume(t *testing.T) {
	handler, _ := newTestHandler(t)
	handler.Start()
	defer handler.Stop()

	p := types.GrandpaPause{
		Delay: 3,
	}

	var digest = types.NewGrandpaConsensusDigest()
	err := digest.Set(p)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.handleConsensusDigest(d, nil)
	require.NoError(t, err)
	nextPause, err := handler.grandpaState.(*state.GrandpaState).GetNextPause()
	require.NoError(t, err)
	require.Equal(t, uint(p.Delay), nextPause)

	headers, _ := state.AddBlocksToState(t, handler.blockState.(*state.BlockState), 3, false)
	for i, h := range headers {
		handler.blockState.(*state.BlockState).SetFinalisedHash(h.Hash(), uint64(i), 0)
	}

	time.Sleep(time.Millisecond * 100)
	require.Nil(t, handler.grandpaPause)

	r := types.GrandpaResume{
		Delay: 3,
	}

	var digest2 = types.NewGrandpaConsensusDigest()
	err = digest2.Set(r)
	require.NoError(t, err)

	data, err = scale.Marshal(digest2)
	require.NoError(t, err)

	d = &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.handleConsensusDigest(d, nil)
	require.NoError(t, err)

	state.AddBlocksToState(t, handler.blockState.(*state.BlockState), 3, false)
	time.Sleep(time.Millisecond * 110)
	require.Nil(t, handler.grandpaResume)

	nextResume, err := handler.grandpaState.(*state.GrandpaState).GetNextResume()
	require.NoError(t, err)
	require.Equal(t, uint(r.Delay+p.Delay), nextResume)
}

func TestNextGrandpaAuthorityChange_OneChange(t *testing.T) {
	handler, _ := newTestHandler(t)
	handler.Start()
	defer handler.Stop()

	const block uint = 3
	sc := types.GrandpaScheduledChange{
		Auths: []types.GrandpaAuthoritiesRaw{},
		Delay: uint32(block),
	}

	var digest = types.NewGrandpaConsensusDigest()
	err := digest.Set(sc)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}
	header := &types.Header{
		Number: 1,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	next := handler.NextGrandpaAuthorityChange()
	require.Equal(t, block, next)

	nextSetID := uint64(1)
	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestNextGrandpaAuthorityChange_MultipleChanges(t *testing.T) {
	handler, _ := newTestHandler(t)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	later := uint32(6)
	sc := types.GrandpaScheduledChange{
		Auths: []types.GrandpaAuthoritiesRaw{},
		Delay: later,
	}

	var digest = types.NewGrandpaConsensusDigest()
	err = digest.Set(sc)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: 1,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	nextSetID := uint64(1)
	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)

	const earlier uint = 4
	fc := types.GrandpaForcedChange{
		Auths: []types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: uint32(earlier),
	}

	digest = types.NewGrandpaConsensusDigest()
	err = digest.Set(fc)
	require.NoError(t, err)

	data, err = scale.Marshal(digest)
	require.NoError(t, err)

	d = &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	next := handler.NextGrandpaAuthorityChange()
	require.Equal(t, earlier+1, next)

	auths, err = handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err = types.NewGrandpaVotersFromAuthoritiesRaw(fc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestHandler_HandleBABEOnDisabled(t *testing.T) {
	handler, _ := newTestHandler(t)
	header := &types.Header{
		Number: 1,
	}

	var digest = types.NewBabeConsensusDigest()
	err := digest.Set(types.BABEOnDisabled{
		ID: 7,
	})
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)
}

func createHeaderWithPreDigest(t *testing.T, slotNumber uint64) *types.Header {
	t.Helper()

	babeHeader := types.NewBabeDigest()
	err := babeHeader.Set(*types.NewBabePrimaryPreDigest(0, slotNumber, [32]byte{}, [64]byte{}))
	require.NoError(t, err)

	enc, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	d := &types.PreRuntimeDigest{
		Data: enc,
	}
	digest := types.NewDigest()
	err = digest.Add(*d)
	require.NoError(t, err)

	return &types.Header{
		Number: 1,
		Digest: digest,
	}
}

func TestHandler_HandleNextEpochData(t *testing.T) {
	expectedDigestBytes := common.MustHexToBytes("0x0108d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01000000000000008eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a4801000000000000004d58630000000000000000000000000000000000000000000000000000000000") //nolint:lll

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	authA := types.AuthorityRaw{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	authB := types.AuthorityRaw{
		Key:    keyring.Bob().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	nextEpochData := types.NextEpochData{
		Authorities: []types.AuthorityRaw{authA, authB},
		Randomness:  [32]byte{77, 88, 99},
	}

	digest := types.NewBabeConsensusDigest()
	err = digest.Set(nextEpochData)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	require.Equal(t, expectedDigestBytes, data)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	header := createHeaderWithPreDigest(t, 10)
	handler, stateSrv := newTestHandler(t)

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	const targetEpoch = 1

	blockHeaderKey := append([]byte("hdr"), header.Hash().ToBytes()...)
	blockHeaderKey = append([]byte("block"), blockHeaderKey...)
	err = stateSrv.DB().Put(blockHeaderKey, []byte{})
	require.NoError(t, err)

	handler.finalised = make(chan *types.FinalisationInfo, 1)

	const timeout = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	handler.finalised <- &types.FinalisationInfo{
		Header: *header,
		Round:  1,
		SetID:  1,
	}

	handler.handleBlockFinalisation(ctx)

	// Before check the epoch data was stored
	// we need to wait for both handle functions to finish
	<-ctx.Done()

	stored, err := handler.epochState.(*state.EpochState).GetEpochData(targetEpoch, nil)
	require.NoError(t, err)

	act, ok := digest.Value().(types.NextEpochData)
	if !ok {
		t.Fatal()
	}

	res, err := act.ToEpochData()
	require.NoError(t, err)
	require.Equal(t, res, stored)
}

func TestHandler_HandleNextConfigData(t *testing.T) {
	var digest = types.NewBabeConsensusDigest()
	nextConfigData := types.NextConfigData{
		C1:             1,
		C2:             8,
		SecondarySlots: 1,
	}

	err := digest.Set(nextConfigData)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	header := createHeaderWithPreDigest(t, 10)

	handler, stateSrv := newTestHandler(t)

	err = handler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	const targetEpoch = 1

	blockHeaderKey := append([]byte("hdr"), header.Hash().ToBytes()...)
	blockHeaderKey = append([]byte("block"), blockHeaderKey...)
	err = stateSrv.DB().Put(blockHeaderKey, []byte{})
	require.NoError(t, err)

	handler.finalised = make(chan *types.FinalisationInfo, 1)

	const timeout = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	handler.finalised <- &types.FinalisationInfo{
		Header: *header,
		Round:  1,
		SetID:  1,
	}

	handler.handleBlockFinalisation(ctx)

	// Before check the config data was stored
	// we need to wait for both handle functions finish
	<-ctx.Done()

	act, ok := digest.Value().(types.NextConfigData)
	if !ok {
		t.Fatal()
	}

	stored, err := handler.epochState.(*state.EpochState).GetConfigData(targetEpoch, nil)
	require.NoError(t, err)
	require.Equal(t, act.ToConfigData(), stored)
}
