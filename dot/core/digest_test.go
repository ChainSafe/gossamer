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

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	. "github.com/ChainSafe/gossamer/dot/core/mocks"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestDigestHandler(t *testing.T, withBABE, withGrandpa bool) *DigestHandler { //nolint
	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	config := state.Config{
		Path:     testDatadirPath,
		LogLevel: log.LvlInfo,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newTestGenesisWithTrieAndHeader(t)
	err = stateSrvc.Initialise(gen, genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	var bp *MockBlockProducer
	if withBABE {
		bp = new(MockBlockProducer)
		blockC := make(chan types.Block)
		bp.On("GetBlockChannel", nil).Return(blockC)
	}

	verifier := new(MockVerifier)
	verifier.On("SetOnDisabled", mock.Anything, mock.Anything).Return(nil)

	dh, err := NewDigestHandler(stateSrvc.Block, stateSrvc.Epoch, stateSrvc.Grandpa, nil, nil)
	require.NoError(t, err)
	return dh
}

func TestDigestHandler_GrandpaScheduledChange(t *testing.T) {
	handler := newTestDigestHandler(t, false, true)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	sc := &types.GrandpaScheduledChange{
		Auths: []*types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: 3,
	}

	data, err := sc.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: big.NewInt(1),
	}

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	headers := addTestBlocksToState(t, 2, handler.blockState)
	for _, h := range headers {
		handler.blockState.SetFinalizedHash(h.Hash(), 0, 0)
	}

	// authorities should change on start of block 3 from start
	headers = addTestBlocksToState(t, 1, handler.blockState)
	for _, h := range headers {
		handler.blockState.SetFinalizedHash(h.Hash(), 0, 0)
	}

	time.Sleep(time.Millisecond * 100)
	setID, err := handler.grandpaState.(*state.GrandpaState).GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, uint64(1), setID)

	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(setID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestDigestHandler_GrandpaForcedChange(t *testing.T) {
	handler := newTestDigestHandler(t, false, true)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	fc := &types.GrandpaForcedChange{
		Auths: []*types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: 3,
	}

	data, err := fc.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: big.NewInt(1),
	}

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	addTestBlocksToState(t, 3, handler.blockState)

	// authorities should change on start of block 4 from start
	addTestBlocksToState(t, 1, handler.blockState)
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

func TestDigestHandler_GrandpaPauseAndResume(t *testing.T) {
	handler := newTestDigestHandler(t, false, true)
	handler.Start()
	defer handler.Stop()

	p := &types.GrandpaPause{
		Delay: 3,
	}

	data, err := p.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.HandleConsensusDigest(d, nil)
	require.NoError(t, err)
	nextPause, err := handler.grandpaState.(*state.GrandpaState).GetNextPause()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(int64(p.Delay)), nextPause)

	headers := addTestBlocksToState(t, 3, handler.blockState)
	for _, h := range headers {
		handler.blockState.SetFinalizedHash(h.Hash(), 0, 0)
	}

	time.Sleep(time.Millisecond * 100)
	require.Nil(t, handler.grandpaPause)

	r := &types.GrandpaResume{
		Delay: 3,
	}

	data, err = r.Encode()
	require.NoError(t, err)

	d = &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.HandleConsensusDigest(d, nil)
	require.NoError(t, err)

	addTestBlocksToState(t, 3, handler.blockState)
	time.Sleep(time.Millisecond * 110)
	require.Nil(t, handler.grandpaResume)

	nextResume, err := handler.grandpaState.(*state.GrandpaState).GetNextResume()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(int64(r.Delay)+int64(p.Delay)), nextResume)
}

func TestNextGrandpaAuthorityChange_OneChange(t *testing.T) {
	handler := newTestDigestHandler(t, false, true)
	handler.Start()
	defer handler.Stop()

	block := uint32(3)
	sc := &types.GrandpaScheduledChange{
		Auths: []*types.GrandpaAuthoritiesRaw{},
		Delay: block,
	}

	data, err := sc.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}
	header := &types.Header{
		Number: big.NewInt(1),
	}

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	next := handler.NextGrandpaAuthorityChange()
	require.Equal(t, uint64(block), next)

	nextSetID := uint64(1)
	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestNextGrandpaAuthorityChange_MultipleChanges(t *testing.T) {
	handler := newTestDigestHandler(t, false, true)
	handler.Start()
	defer handler.Stop()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	later := uint32(6)
	sc := &types.GrandpaScheduledChange{
		Auths: []*types.GrandpaAuthoritiesRaw{},
		Delay: later,
	}

	data, err := sc.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	header := &types.Header{
		Number: big.NewInt(1),
	}

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	nextSetID := uint64(1)
	auths, err := handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err := types.NewGrandpaVotersFromAuthoritiesRaw(sc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)

	earlier := uint32(4)
	fc := &types.GrandpaForcedChange{
		Auths: []*types.GrandpaAuthoritiesRaw{
			{Key: kr.Alice().Public().(*ed25519.PublicKey).AsBytes(), ID: 0},
		},
		Delay: earlier,
	}

	data, err = fc.Encode()
	require.NoError(t, err)

	d = &types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              data,
	}

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	next := handler.NextGrandpaAuthorityChange()
	require.Equal(t, uint64(earlier+1), next)

	auths, err = handler.grandpaState.(*state.GrandpaState).GetAuthorities(nextSetID)
	require.NoError(t, err)
	expected, err = types.NewGrandpaVotersFromAuthoritiesRaw(fc.Auths)
	require.NoError(t, err)
	require.Equal(t, expected, auths)
}

func TestDigestHandler_HandleBABEOnDisabled(t *testing.T) {
	handler := newTestDigestHandler(t, true, false)

	babemock := new(MockBlockProducer)
	babemock.On("SetOnDisabled", uint32(7))

	header := &types.Header{
		Number: big.NewInt(1),
	}

	verifier := new(MockVerifier)
	verifier.On("SetOnDisabled", uint32(7), header).Return(nil)

	handler.babe = babemock
	handler.verifier = verifier

	digest := &types.BABEOnDisabled{
		ID: 7,
	}

	data, err := digest.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	err = handler.HandleConsensusDigest(d, header)

	require.NoError(t, err)

	babemock.AssertCalled(t, "SetOnDisabled", uint32(7))
}

func createHeaderWithPreDigest(slotNumber uint64) *types.Header {
	babeHeader := types.NewBabePrimaryPreDigest(0, slotNumber, [32]byte{}, [64]byte{})

	enc := babeHeader.Encode()
	digest := &types.PreRuntimeDigest{
		Data: enc,
	}

	return &types.Header{
		Digest: types.Digest{digest},
	}
}

func TestDigestHandler_HandleNextEpochData(t *testing.T) {
	handler := newTestDigestHandler(t, true, false)
	handler.Start()
	defer handler.Stop()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	authA := &types.AuthorityRaw{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	authB := &types.AuthorityRaw{
		Key:    keyring.Bob().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	digest := &types.NextEpochData{
		Authorities: []*types.AuthorityRaw{authA, authB},
		Randomness:  [32]byte{77, 88, 99},
	}

	data, err := digest.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	header := createHeaderWithPreDigest(10)

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	stored, err := handler.epochState.(*state.EpochState).GetEpochData(1)
	require.NoError(t, err)
	res, err := digest.ToEpochData()
	require.NoError(t, err)
	require.Equal(t, res, stored)
}

func TestDigestHandler_HandleNextConfigData(t *testing.T) {
	handler := newTestDigestHandler(t, true, false)
	handler.Start()
	defer handler.Stop()

	digest := &types.NextConfigData{
		C1:             1,
		C2:             8,
		SecondarySlots: 1,
	}

	data, err := digest.Encode()
	require.NoError(t, err)

	d := &types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              data,
	}

	header := createHeaderWithPreDigest(10)

	err = handler.HandleConsensusDigest(d, header)
	require.NoError(t, err)

	stored, err := handler.epochState.(*state.EpochState).GetConfigData(1)
	require.NoError(t, err)
	require.Equal(t, digest.ToConfigData(), stored)
}
