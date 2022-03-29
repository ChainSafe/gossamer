// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

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

	err = gs.IncrementSetID()
	require.NoError(t, err)

	setID, err := gs.GetCurrentSetID()
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

	err = gs.IncrementSetID()
	require.NoError(t, err)

	setID, err = gs.GetSetIDByBlockNumber(100)
	require.NoError(t, err)
	require.Equal(t, genesisSetID, setID)

	setID, err = gs.GetSetIDByBlockNumber(101)
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
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

func TestImportGrandpaChangesKeepDecreasingOrdered(t *testing.T) {
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil)
	require.NoError(t, err)

	scheduledChanges := types.GrandpaScheduledChange{}
	forkChainA := issueBlocksWithGRANDPAScheduledChanges(t, keyring.KeyAlice, blockState,
		testGenesisHeader, 3)

	forkChainA = shuffleHeaderSlice(forkChainA)

	for _, header := range forkChainA {
		grandpaConsensusDigest := types.NewGrandpaConsensusDigest()
		err := grandpaConsensusDigest.Set(scheduledChanges)
		require.NoError(t, err)

		err = gs.ImportGrandpaChange(header, grandpaConsensusDigest)
		require.NoError(t, err)
	}

	require.Len(t, gs.forks, 1)

	forkAStartHash := forkChainA[0].Hash()
	linkedList := gs.forks[forkAStartHash]

	for linkedList.Next != nil {
		require.Greater(t,
			linkedList.header.Number, linkedList.Next.header.Number)
		linkedList = linkedList.Next
	}

	fmt.Println()
}

func issueBlocksWithGRANDPAScheduledChanges(t *testing.T, kp *sr25519.Keypair,
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
	headers = append(headers, issueBlocksWithGRANDPAScheduledChanges(t, kp, bs, header, size-1)...)
	return headers
}

func shuffleHeaderSlice(headers []*types.Header) []*types.Header {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(headers), func(i, j int) {
		headers[i], headers[j] = headers[j], headers[i]
	})
	return headers
}
