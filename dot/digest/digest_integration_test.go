//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"testing"
	"time"

	"context"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*Handler, *BlockImportHandler, *state.Service) {
	testDatadirPath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		Telemetry: telemetryMock,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&gen, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	dh, err := NewHandler(stateSrvc.Block, stateSrvc.Epoch)
	require.NoError(t, err)

	blockImportHandler := NewBlockImportHandler(stateSrvc.Epoch)
	return dh, blockImportHandler, stateSrvc
}

func TestHandler_HandleBABEOnDisabled(t *testing.T) {
	_, blockImportHandler, _ := newTestHandler(t)
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

	err = blockImportHandler.handleConsensusDigest(d, header)
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
	handler, blockImportHandler, stateSrv := newTestHandler(t)

	err = blockImportHandler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	const targetEpoch = 1

	blockHeaderKey := append([]byte("hdr"), header.Hash().ToBytes()...)
	blockHeaderKey = append([]byte("block"), blockHeaderKey...)
	err = stateSrv.DB().Put(blockHeaderKey, []byte{})
	require.NoError(t, err)

	handler.finalised = make(chan *types.FinalisationInfo, 1)

	const timeout = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	handler.finalised <- &types.FinalisationInfo{
		Header: *header,
		Round:  1,
		SetID:  1,
	}

	handler.handleBlockFinalisation(ctx)

	stored, err := handler.epochState.(*state.EpochState).GetEpochData(targetEpoch, nil)
	require.NoError(t, err)

	digestValue, err := digest.Value()
	require.NoError(t, err)
	act, ok := digestValue.(types.NextEpochData)
	if !ok {
		t.Fatal()
	}

	res, err := act.ToEpochData()
	require.NoError(t, err)
	require.Equal(t, res, stored)
}

func TestHandler_HandleNextConfigData(t *testing.T) {
	var digest = types.NewBabeConsensusDigest()
	nextConfigData := types.NextConfigDataV1{
		C1:             1,
		C2:             8,
		SecondarySlots: 1,
	}

	versionedNextConfigData := types.NewVersionedNextConfigData()
	versionedNextConfigData.Set(nextConfigData)

	err := digest.Set(versionedNextConfigData)
	require.NoError(t, err)

	data, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	header := createHeaderWithPreDigest(t, 10)

	handler, blockImportHandler, stateSrv := newTestHandler(t)

	err = blockImportHandler.handleConsensusDigest(d, header)
	require.NoError(t, err)

	const targetEpoch = 1

	blockHeaderKey := append([]byte("hdr"), header.Hash().ToBytes()...)
	blockHeaderKey = append([]byte("block"), blockHeaderKey...)
	err = stateSrv.DB().Put(blockHeaderKey, []byte{})
	require.NoError(t, err)

	handler.finalised = make(chan *types.FinalisationInfo, 1)

	const timeout = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	handler.finalised <- &types.FinalisationInfo{
		Header: *header,
		Round:  1,
		SetID:  1,
	}

	handler.handleBlockFinalisation(ctx)

	digestValue, err := digest.Value()
	require.NoError(t, err)
	nextVersionedConfigData, ok := digestValue.(types.VersionedNextConfigData)
	if !ok {
		t.Fatal()
	}

	decodedNextConfigData, err := nextVersionedConfigData.Value()
	require.NoError(t, err)

	decodedNextConfigDataV1, ok := decodedNextConfigData.(types.NextConfigDataV1)
	if !ok {
		t.Fatal()
	}

	stored, err := handler.epochState.(*state.EpochState).GetConfigData(targetEpoch, nil)
	require.NoError(t, err)
	require.Equal(t, decodedNextConfigDataV1.ToConfigData(), stored)
}
