// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/golang/mock/gomock"
	"github.com/gtank/merlin"

	"github.com/stretchr/testify/require"
)

var (
	kr, _     = keystore.NewEd25519Keyring()
	testAuths = []types.GrandpaVoter{
		{Key: *kr.Alice().Public().(*ed25519.PublicKey), ID: 0},
	}
)

func TestNewGrandpaStateFromGenesis(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths)
	require.NoError(t, err)

	currSetID, err := gs.GetCurrentSetID()
	require.NoError(t, err)
	require.Equal(t, genesisSetID, currSetID)

	auths, err := gs.GetAuthorities(currSetID)
	require.NoError(t, err)
	require.Equal(t, testAuths, auths)

	num, err := gs.GetSetIDChange(0)
	require.NoError(t, err)
	require.Equal(t, uint(0), num)
}

func TestGrandpaState_SetNextChange(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths)
	require.NoError(t, err)

	err = gs.SetNextChange(testAuths, 1)
	require.NoError(t, err)

	auths, err := gs.GetAuthorities(genesisSetID + 1)
	require.NoError(t, err)
	require.Equal(t, testAuths, auths)

	atBlock, err := gs.GetSetIDChange(genesisSetID + 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), atBlock)
}

func TestGrandpaState_IncrementSetID(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths)
	require.NoError(t, err)

	setID, err := gs.IncrementSetID()
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
}

func TestGrandpaState_GetSetIDByBlockNumber(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths)
	require.NoError(t, err)

	err = gs.SetNextChange(testAuths, 100)
	require.NoError(t, err)

	setID, err := gs.GetSetIDByBlockNumber(50)
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(100)
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(101)
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)

	newSetID, err := gs.IncrementSetID()
	require.NoError(t, err)

	setID, err = gs.GetSetIDByBlockNumber(100)
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(101)
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
	require.Equal(t, genesisSetID+1, newSetID)
}

func TestGrandpaState_LatestRound(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths)
	require.NoError(t, err)

	r, err := gs.GetLatestRound()
	require.NoError(t, err)
	require.Equal(t, uint64(0), r)

	err = gs.SetLatestRound(99)
	require.NoError(t, err)

	r, err = gs.GetLatestRound()
	require.NoError(t, err)
	require.Equal(t, uint64(99), r)
}

func testBlockState(t *testing.T, db chaindb.Database) *BlockState {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockClient(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
	header := testGenesisHeader

	bs, err := NewBlockStateFromGenesis(db, newTriesEmpty(), header, telemetryMock)
	require.NoError(t, err)

	// loads in-memory tries with genesis state root, should be deleted
	// after another block is finalised
	tr := trie.NewEmptyTrie()
	err = tr.Load(bs.db, header.StateRoot)
	require.NoError(t, err)
	bs.tries.softSet(header.StateRoot, tr)

	return bs
}

func TestForcedScheduledChangesOrder(t *testing.T) {
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil)
	require.NoError(t, err)

	aliceHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, gs.blockState,
		testGenesisHeader, 5)

	bobHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyBob, gs.blockState,
		aliceHeaders[1], 5)

	charlieHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, gs.blockState,
		aliceHeaders[2], 6)

	forcedChanges := map[*types.Header]types.GrandpaForcedChange{
		bobHeaders[1]: {
			Delay: 1,
			Auths: []types.GrandpaAuthoritiesRaw{
				{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
			},
		},
		aliceHeaders[3]: {
			Delay: 5,
			Auths: []types.GrandpaAuthoritiesRaw{
				{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
			},
		},
		charlieHeaders[4]: {
			Delay: 3,
			Auths: []types.GrandpaAuthoritiesRaw{
				{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
			},
		},
	}

	for header, fc := range forcedChanges {
		err := gs.addForcedChange(header, fc)
		require.NoError(t, err, "failed to add forced change")
	}

	for idx := 0; idx < len(gs.forcedChanges)-1; idx++ {
		currentChange := gs.forcedChanges[idx]
		nextChange := gs.forcedChanges[idx+1]

		require.LessOrEqual(t, currentChange.effectiveNumber(),
			nextChange.effectiveNumber())

		require.LessOrEqual(t, currentChange.announcingHeader.Number,
			nextChange.announcingHeader.Number)
	}
}

func TestShouldNotAddMoreThanOneForcedChangeInTheSameFork(t *testing.T) {
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil)
	require.NoError(t, err)

	aliceHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, gs.blockState,
		testGenesisHeader, 5)

	bobHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyBob, gs.blockState,
		aliceHeaders[1], 5)

	someForcedChange := types.GrandpaForcedChange{
		Delay: 1,
		Auths: []types.GrandpaAuthoritiesRaw{
			{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
			{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
			{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
		},
	}

	// adding more than one forced changes in the same branch
	err = gs.addForcedChange(aliceHeaders[3], someForcedChange)
	require.NoError(t, err)

	err = gs.addForcedChange(aliceHeaders[4], someForcedChange)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAlreadyHasForcedChanges)

	// adding the same forced change twice
	err = gs.addForcedChange(bobHeaders[2], someForcedChange)
	require.NoError(t, err)

	err = gs.addForcedChange(bobHeaders[2], someForcedChange)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicatedHashes)
}

func issueBlocksWithBABEPrimary(t *testing.T, kp *sr25519.Keypair,
	bs *BlockState, parentHeader *types.Header, size int) (headers []*types.Header) {
	t.Helper()

	transcript := merlin.NewTranscript("BABE") //string(types.BabeEngineID[:])
	crypto.AppendUint64(transcript, []byte("slot number"), 1)
	crypto.AppendUint64(transcript, []byte("current epoch"), 1)
	transcript.AppendMessage([]byte("chain randomness"), []byte{})

	output, proof, err := kp.VrfSign(transcript)
	require.NoError(t, err)

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := types.NewDigest()

	require.NoError(t, digest.Add(*preRuntimeDigest))
	header := &types.Header{
		ParentHash: parentHeader.Hash(),
		Number:     parentHeader.Number + 1,
		Digest:     digest,
	}

	block := &types.Block{
		Header: *header,
		Body:   *types.NewBody([]types.Extrinsic{}),
	}

	err = bs.AddBlock(block)
	require.NoError(t, err)

	if size <= 0 {
		headers = append(headers, header)
		return headers
	}

	headers = append(headers, header)
	headers = append(headers, issueBlocksWithBABEPrimary(t, kp, bs, header, size-1)...)
	return headers
}
