// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
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

func TestAddScheduledChangesKeepTheRightForkTree(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil)

	/*
	* create chainA and two forks: chainB and chainC
	*
	*      / -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (B)
	* 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (A)
	*                          \ -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 (C)
	 */
	chainA := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, gs.blockState, testGenesisHeader, 10)
	chainB := issueBlocksWithBABEPrimary(t, keyring.KeyBob, gs.blockState, chainA[1], 9)
	chainC := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, gs.blockState, chainA[5], 10)

	scheduledChange := &types.GrandpaScheduledChange{
		Delay: 0, // delay of 0 means the modifications should be applied immediately
		Auths: []types.GrandpaAuthoritiesRaw{
			{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
			{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
			{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
		},
	}

	// headersToAdd enables tracking error while adding expecific entries
	// to the scheduled change fork tree, eg.
	// - adding duplicate hashes entries: while adding the first entry everything should be ok,
	//   however when adding the second duplicated entry we should expect the errDuplicateHashes error
	type headersToAdd struct {
		header  *types.Header
		wantErr error
	}

	tests := map[string]struct {
		headersWithScheduledChanges []headersToAdd
		expectedRoots               int
		highestFinalizedHeader      *types.Header
	}{
		"add_scheduled_changes_only_with_roots": {
			headersWithScheduledChanges: []headersToAdd{
				{header: chainA[6]},
				{header: chainB[3]},
			},
			expectedRoots: 2,
		},
		"add_scheduled_changes_with_roots_and_children": {
			headersWithScheduledChanges: []headersToAdd{
				{header: chainA[6]}, {header: chainA[8]},
				{header: chainB[3]}, {header: chainB[7]}, {header: chainB[9]},
				{header: chainC[8]},
			},
			expectedRoots: 3,
		},
		"add_scheduled_change_before_highest_finalized_header": {
			headersWithScheduledChanges: []headersToAdd{
				{header: chainA[3], wantErr: ErrLowerThanBestFinalized},
			},
			highestFinalizedHeader: chainA[5],
			expectedRoots:          0,
		},
		"add_scheduled_changes_with_same_hash": {
			headersWithScheduledChanges: []headersToAdd{
				{header: chainA[3]},
				{header: chainA[3], wantErr: fmt.Errorf("could not import scheduled change: %w",
					errDuplicateHashes)},
			},
			expectedRoots: 0,
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			// clear the scheduledChangeRoots after the test ends
			// this does not cause race condition because t.Run without
			// t.Parallel() blocks until this function returns
			defer func() {
				gs.scheduledChangeRoots = gs.scheduledChangeRoots[:0]
			}()

			updateHighestFinalizedHeaderOrDefault(t, gs.blockState, tt.highestFinalizedHeader, chainA[0])

			for _, entry := range tt.headersWithScheduledChanges {
				err := gs.addScheduledChange(entry.header, *scheduledChange)

				if entry.wantErr != nil {
					require.Error(t, err)
					require.EqualError(t, err, entry.wantErr.Error())
					return
				} else {
					require.NoError(t, err)
				}
			}

			require.Len(t, gs.scheduledChangeRoots, tt.expectedRoots)

			for _, root := range gs.scheduledChangeRoots {
				parentHash := root.change.announcingHeader.Hash()
				assertDescendantChildren(t, parentHash, gs.blockState.IsDescendantOf, root.nodes)
			}
		})
	}
}

func assertDescendantChildren(t *testing.T, parentHash common.Hash, isDescendantOfFunc isDescendantOfFunc,
	scheduledChanges []*pendingChangeNode) {
	t.Helper()

	for _, scheduled := range scheduledChanges {
		scheduledChangeHash := scheduled.change.announcingHeader.Hash()
		isDescendant, err := isDescendantOfFunc(parentHash, scheduledChangeHash)
		require.NoError(t, err)
		require.Truef(t, isDescendant, "%s is not descendant of %s", scheduledChangeHash, parentHash)

		assertDescendantChildren(t, scheduledChangeHash, isDescendantOfFunc, scheduled.nodes)
	}
}

// updateHighestFinalizedHeaderOrDefault will update the current highest finalized header
// with the value of newHighest, if the newHighest is nil then it will use the def value
func updateHighestFinalizedHeaderOrDefault(t *testing.T, bs *BlockState, newHighest, def *types.Header) {
	t.Helper()

	round, setID, err := bs.GetHighestRoundAndSetID()
	require.NoError(t, err)

	if newHighest != nil {
		bs.db.Put(finalisedHashKey(round, setID), newHighest.Hash().ToBytes())
	} else {
		bs.db.Put(finalisedHashKey(round, setID), def.Hash().ToBytes())
	}
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
	require.ErrorIs(t, err, errDuplicateHashes)
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

func TestNextGrandpaAuthorityChange(t *testing.T) {
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	tests := map[string]struct {
		forcedChange               *types.GrandpaForcedChange
		forcedChangeAnnoucingIndex int

		scheduledChange               *types.GrandpaScheduledChange
		scheduledChangeAnnoucingIndex int

		wantErr             error
		expectedBlockNumber uint
	}{
		"no_forced_change_no_scheduled_change": {
			wantErr: ErrNoChanges,
		},
		"only_forced_change": {
			forcedChangeAnnoucingIndex: 2, // in the chain headers slice the index 2 == block number 3
			forcedChange: &types.GrandpaForcedChange{
				Delay: 2,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			expectedBlockNumber: 5,
		},
		"only_scheduled_change": {
			scheduledChangeAnnoucingIndex: 3, // in the chain headers slice the index 3 == block number 4
			scheduledChange: &types.GrandpaScheduledChange{
				Delay: 4,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			expectedBlockNumber: 8,
		},
		"forced_change_before_scheduled_change": {
			forcedChangeAnnoucingIndex: 2, // in the chain headers slice the index 2 == block number 3
			forcedChange: &types.GrandpaForcedChange{
				Delay: 2,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			scheduledChangeAnnoucingIndex: 3, // in the chain headers slice the index 3 == block number 4
			scheduledChange: &types.GrandpaScheduledChange{
				Delay: 4,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			expectedBlockNumber: 5, // forced change occurs before the scheduled change
		},
		"scheduled_change_before_forced_change": {
			scheduledChangeAnnoucingIndex: 3, // in the chain headers slice the index 3 == block number 4
			scheduledChange: &types.GrandpaScheduledChange{
				Delay: 4,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			forcedChangeAnnoucingIndex: 8, // in the chain headers slice the index 8 == block number 9
			forcedChange: &types.GrandpaForcedChange{
				Delay: 1,
				Auths: []types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				},
			},
			expectedBlockNumber: 8, // scheduled change occurs before the forced change
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			blockState := testBlockState(t, db)

			gs, err := NewGrandpaStateFromGenesis(db, blockState, nil)
			require.NoError(t, err)

			const sizeOfChain = 10

			chainHeaders := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, gs.blockState,
				testGenesisHeader, sizeOfChain)

			if tt.forcedChange != nil {
				gs.addForcedChange(chainHeaders[tt.forcedChangeAnnoucingIndex],
					*tt.forcedChange)
			}

			if tt.scheduledChange != nil {
				gs.addScheduledChange(chainHeaders[tt.scheduledChangeAnnoucingIndex],
					*tt.scheduledChange)
			}

			lastBlockOnChain := chainHeaders[sizeOfChain].Hash()
			blockNumber, err := gs.NextGrandpaAuthorityChange(lastBlockOnChain)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
				require.Zero(t, blockNumber)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedBlockNumber, blockNumber)
			}
		})
	}
}
