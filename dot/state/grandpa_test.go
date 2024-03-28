// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/gtank/merlin"
	"go.uber.org/mock/gomock"

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
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths, nil)
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
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths, nil)
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
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths, nil)
	require.NoError(t, err)

	setID, err := gs.IncrementSetID()
	require.NoError(t, err)
	require.Equal(t, genesisSetID+1, setID)
}

func TestGrandpaState_GetSetIDByBlockNumber(t *testing.T) {
	db := NewInMemoryDB(t)
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths, nil)
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
	gs, err := NewGrandpaStateFromGenesis(db, nil, testAuths, nil)
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

func testBlockState(t *testing.T, db database.Database) *BlockState {
	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.AssignableToTypeOf(&telemetry.NotifyFinalized{}))
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
	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil, nil)
	require.NoError(t, err)

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
		"add_scheduled_changes_with_same_hash": {
			headersWithScheduledChanges: []headersToAdd{
				{header: chainA[3]},
				{
					header: chainA[3],
					wantErr: fmt.Errorf("cannot import scheduled change: %w: %s",
						errDuplicateHashes, chainA[3].Hash()),
				},
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
				gs.scheduledChangeRoots = new(changeTree)
			}()

			updateHighestFinalizedHeaderOrDefault(t, gs.blockState, tt.highestFinalizedHeader, chainA[0])

			for _, entry := range tt.headersWithScheduledChanges {
				err := gs.addScheduledChange(entry.header, *scheduledChange)

				if entry.wantErr != nil {
					require.Error(t, err)
					require.EqualError(t, err, entry.wantErr.Error())
					return
				}

				require.NoError(t, err)
			}

			require.Len(t, *gs.scheduledChangeRoots, tt.expectedRoots)

			for _, root := range *gs.scheduledChangeRoots {
				parentHash := root.change.announcingHeader.Hash()
				assertDescendantChildren(t, parentHash, gs.blockState.IsDescendantOf, root.nodes)
			}
		})
	}
}

func assertDescendantChildren(t *testing.T, parentHash common.Hash, isDescendantOfFunc isDescendantOfFunc,
	changes changeTree) {
	t.Helper()

	for _, scheduled := range changes {
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
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil, nil)
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

	forcedChangesSlice := *gs.forcedChanges
	for idx := 0; idx < gs.forcedChanges.Len()-1; idx++ {
		currentChange := forcedChangesSlice[idx]
		nextChange := forcedChangesSlice[idx+1]

		require.LessOrEqual(t, currentChange.effectiveNumber(),
			nextChange.effectiveNumber())

		require.LessOrEqual(t, currentChange.announcingHeader.Number,
			nextChange.announcingHeader.Number)
	}
}

func TestShouldNotAddMoreThanOneForcedChangeInTheSameFork(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	db := NewInMemoryDB(t)
	blockState := testBlockState(t, db)

	gs, err := NewGrandpaStateFromGenesis(db, blockState, nil, nil)
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
	require.ErrorIs(t, err, errAlreadyHasForcedChange)

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

	transcript := merlin.NewTranscript("BABE")
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
	t.Parallel()

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
			wantErr: ErrNoNextAuthorityChange,
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

			gs, err := NewGrandpaStateFromGenesis(db, blockState, nil, nil)
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

			lastBlockOnChain := chainHeaders[sizeOfChain]
			blockNumber, err := gs.NextGrandpaAuthorityChange(lastBlockOnChain.Hash(), lastBlockOnChain.Number)

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

func TestApplyForcedChanges(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesisGrandpaVoters := []types.GrandpaAuthoritiesRaw{
		{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
	}

	genesisAuths, err := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
	require.NoError(t, err)

	const sizeOfChain = 10
	genericForks := func(t *testing.T, blockState *BlockState) [][]*types.Header {

		/*
		* create chainA and two forks: chainB and chainC
		*
		*      / -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 (B)
		* 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (A)
		*                          \ -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 (C)
		 */
		chainA := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, blockState, testGenesisHeader, sizeOfChain)
		chainB := issueBlocksWithBABEPrimary(t, keyring.KeyBob, blockState, chainA[1], sizeOfChain)
		chainC := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, blockState, chainA[5], sizeOfChain)

		return [][]*types.Header{
			chainA, chainB, chainC,
		}
	}

	tests := map[string]struct {
		wantErr error
		// 2 indexed array where the 0 index describes the fork and the 1 index describes the header
		importedHeader              [2]int
		expectedGRANDPAAuthoritySet []types.GrandpaAuthoritiesRaw
		expectedSetID               uint64
		expectedPruning             bool

		generateForks func(t *testing.T, blockState *BlockState) [][]*types.Header
		changes       func(*GrandpaState, [][]*types.Header)
		telemetryMock *MockTelemetry
	}{
		"no_forced_changes": {
			generateForks:               genericForks,
			importedHeader:              [2]int{0, 3}, // chain A from and header number 4
			expectedSetID:               0,
			expectedGRANDPAAuthoritySet: genesisGrandpaVoters,
			expectedPruning:             false,
			telemetryMock:               nil,
		},
		"apply_forced_change_without_pending_scheduled_changes": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock8 := headers[0][7]
				gs.addForcedChange(chainABlock8, types.GrandpaForcedChange{
					Delay:              2,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainCBlock15 := headers[2][8]
				gs.addForcedChange(chainCBlock15, types.GrandpaForcedChange{
					Delay:              1,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			importedHeader:  [2]int{0, 9}, // import block number 10 from fork A
			expectedSetID:   1,
			expectedPruning: true,
			expectedGRANDPAAuthoritySet: []types.GrandpaAuthoritiesRaw{
				{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
				{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
			},
			telemetryMock: func() *MockTelemetry {
				ctrl := gomock.NewController(t)

				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(gomock.Eq(&telemetry.AfgApplyingForcedAuthoritySetChange{Block: "8"}))

				return telemetryMock
			}(),
		},
		"import_block_before_forced_change_should_do_nothing": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainCBlock9 := headers[2][2]
				gs.addForcedChange(chainCBlock9, types.GrandpaForcedChange{
					Delay:              3,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			importedHeader:              [2]int{2, 1}, // import block number 7 from chain C
			expectedSetID:               0,
			expectedPruning:             false,
			expectedGRANDPAAuthoritySet: genesisGrandpaVoters,
			telemetryMock:               nil,
		},
		"import_block_from_another_fork_should_do_nothing": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainCBlock9 := headers[2][2]
				gs.addForcedChange(chainCBlock9, types.GrandpaForcedChange{
					Delay:              3,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			importedHeader:              [2]int{1, 9}, // import block number 12 from chain B
			expectedSetID:               0,
			expectedPruning:             false,
			expectedGRANDPAAuthoritySet: genesisGrandpaVoters,
			telemetryMock:               nil,
		},
		"apply_forced_change_with_pending_scheduled_changes_should_fail": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainBBlock6 := headers[1][3]
				gs.addScheduledChange(chainBBlock6, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainCBlock9 := headers[2][2]
				gs.addForcedChange(chainCBlock9, types.GrandpaForcedChange{
					Delay:              3,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock9 := headers[1][6]
				gs.addForcedChange(chainBBlock9, types.GrandpaForcedChange{
					Delay:              2,
					BestFinalizedBlock: 6,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			importedHeader:              [2]int{1, 8}, // block number 11 imported
			wantErr:                     errPendingScheduledChanges,
			expectedGRANDPAAuthoritySet: genesisGrandpaVoters,
			expectedSetID:               0,
			expectedPruning:             false,
			telemetryMock:               nil,
		},
		"apply_forced_change_should_prune_scheduled_changes": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainBBlock12 := headers[1][9]
				chainBBlock11 := headers[1][8]
				chainBBlock10 := headers[1][7]

				// add scheduled changes for block 10, 11 and 12 from for B
				addScheduledChanges := []*types.Header{chainBBlock10, chainBBlock11, chainBBlock12}
				for _, blockHeader := range addScheduledChanges {
					gs.addScheduledChange(blockHeader, types.GrandpaScheduledChange{
						Delay: 0,
						Auths: []types.GrandpaAuthoritiesRaw{
							{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
						},
					})
				}

				// add a forced change for block 9 from chain C with delay of 3 blocks
				// once block 11 got imported Gossamer should clean up all the scheduled + forced changes
				chainCBlock9 := headers[2][2]
				gs.addForcedChange(chainCBlock9, types.GrandpaForcedChange{
					Delay:              3,
					BestFinalizedBlock: 2,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			importedHeader:              [2]int{2, 7}, // import block 12 from chain C
			expectedGRANDPAAuthoritySet: genesisGrandpaVoters,
			expectedSetID:               0,
			expectedPruning:             false,
			telemetryMock:               nil,
		},
	}

	for tname, tt := range tests {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			blockState := testBlockState(t, db)

			voters := types.NewGrandpaVotersFromAuthorities(genesisAuths)
			gs, err := NewGrandpaStateFromGenesis(db, blockState, voters, tt.telemetryMock)
			require.NoError(t, err)

			forks := tt.generateForks(t, blockState)
			if tt.changes != nil {
				tt.changes(gs, forks)
			}

			selectedFork := forks[tt.importedHeader[0]]
			selectedImportedHeader := selectedFork[tt.importedHeader[1]]

			amountOfForced := gs.forcedChanges.Len()
			amountOfScheduled := gs.scheduledChangeRoots.Len()

			err = gs.ApplyForcedChanges(selectedImportedHeader)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.expectedPruning {
				// we should reset the changes set once a forced change is applied
				const expectedLen = 0
				require.Equal(t, expectedLen, gs.forcedChanges.Len())
				require.Equal(t, expectedLen, gs.scheduledChangeRoots.Len())
			} else {
				require.Equal(t, amountOfForced, gs.forcedChanges.Len())
				require.Equal(t, amountOfScheduled, gs.scheduledChangeRoots.Len())
			}

			currentSetID, err := gs.GetCurrentSetID()
			require.NoError(t, err)
			require.Equal(t, tt.expectedSetID, currentSetID)

			expectedAuths, err := types.GrandpaAuthoritiesRawToAuthorities(tt.expectedGRANDPAAuthoritySet)
			require.NoError(t, err)
			expectedVoters := types.NewGrandpaVotersFromAuthorities(expectedAuths)

			gotVoters, err := gs.GetAuthorities(tt.expectedSetID)
			require.NoError(t, err)

			require.Equal(t, expectedVoters, gotVoters)
		})
	}
}

func TestApplyScheduledChangesKeepDescendantForcedChanges(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesisGrandpaVoters := []types.GrandpaAuthoritiesRaw{
		{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
	}

	genesisAuths, err := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
	require.NoError(t, err)

	const sizeOfChain = 10
	genericForks := func(t *testing.T, blockState *BlockState) [][]*types.Header {

		/*
		* create chainA and two forks: chainB and chainC
		*
		*      / -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 (B)
		* 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (A)
		*                          \ -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 (C)
		 */
		chainA := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, blockState, testGenesisHeader, sizeOfChain)
		chainB := issueBlocksWithBABEPrimary(t, keyring.KeyBob, blockState, chainA[1], sizeOfChain)
		chainC := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, blockState, chainA[5], sizeOfChain)

		return [][]*types.Header{
			chainA, chainB, chainC,
		}
	}

	tests := map[string]struct {
		finalizedHeader [2]int // 2 index array where the 0 index describes the fork and the 1 index describes the header

		generateForks func(*testing.T, *BlockState) [][]*types.Header
		changes       func(*GrandpaState, [][]*types.Header)

		wantErr error

		expectedForcedChangesLen int
	}{
		"no_forced_changes": {
			generateForks:            genericForks,
			expectedForcedChangesLen: 0,
		},
		"finalized_hash_should_keep_descendant_forced_changes": {
			generateForks:            genericForks,
			expectedForcedChangesLen: 1,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock6 := headers[0][5]
				gs.addForcedChange(chainABlock6, types.GrandpaForcedChange{
					Delay:              1,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock6 := headers[1][3]
				gs.addForcedChange(chainBBlock6, types.GrandpaForcedChange{
					Delay:              2,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader: [2]int{0, 3}, // finalize header number 4 from chain A
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			blockState := testBlockState(t, db)

			voters := types.NewGrandpaVotersFromAuthorities(genesisAuths)
			gs, err := NewGrandpaStateFromGenesis(db, blockState, voters, nil)
			require.NoError(t, err)

			forks := tt.generateForks(t, gs.blockState)

			if tt.changes != nil {
				tt.changes(gs, forks)
			}

			selectedFork := forks[tt.finalizedHeader[0]]
			selectedFinalizedHeader := selectedFork[tt.finalizedHeader[1]]

			err = gs.forcedChanges.pruneChanges(selectedFinalizedHeader.Hash(), gs.blockState.IsDescendantOf)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)

				require.Len(t, *gs.forcedChanges, tt.expectedForcedChangesLen)

				for _, forcedChange := range *gs.forcedChanges {
					isDescendant, err := gs.blockState.IsDescendantOf(
						selectedFinalizedHeader.Hash(), forcedChange.announcingHeader.Hash())

					require.NoError(t, err)
					require.True(t, isDescendant)
				}
			}
		})
	}
}

func TestApplyScheduledChangeGetApplicableChange(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesisGrandpaVoters := []types.GrandpaAuthoritiesRaw{
		{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
	}

	genesisAuths, err := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
	require.NoError(t, err)

	const sizeOfChain = 10
	genericForks := func(t *testing.T, blockState *BlockState) [][]*types.Header {
		/*
		* create chainA and two forks: chainB and chainC
		*
		*      / -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 (B)
		* 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (A)
		*                          \ -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 (C)
		 */
		chainA := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, blockState, testGenesisHeader, sizeOfChain)
		chainB := issueBlocksWithBABEPrimary(t, keyring.KeyBob, blockState, chainA[1], sizeOfChain)
		chainC := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, blockState, chainA[5], sizeOfChain)

		return [][]*types.Header{
			chainA, chainB, chainC,
		}
	}

	tests := map[string]struct {
		finalizedHeader                 [2]int
		changes                         func(*GrandpaState, [][]*types.Header)
		generateForks                   func(*testing.T, *BlockState) [][]*types.Header
		wantErr                         error
		expectedChange                  *pendingChange
		expectedScheduledChangeRootsLen int
	}{
		"empty_scheduled_changes": {
			generateForks:   genericForks,
			finalizedHeader: [2]int{0, 1}, // finalized block from chainA header number 2
		},
		"scheduled_change_being_finalized_should_be_applied": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock6 := headers[0][5]
				gs.addScheduledChange(chainABlock6, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			expectedChange: &pendingChange{
				delay: 0,
				nextAuthorities: func() []types.Authority {
					auths, _ := types.GrandpaAuthoritiesRawToAuthorities(
						[]types.GrandpaAuthoritiesRaw{
							{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
							{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
							{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
						},
					)
					return auths
				}(),
			},
			finalizedHeader: [2]int{0, 5}, // finalize block number 6 from chain A
		},
		"apply_change_and_update_scheduled_changes_with_the_children": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainBBlock4 := headers[1][1] // block number 4 from chain B
				gs.addScheduledChange(chainBBlock4, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyFerdie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyGeorge.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock7 := headers[1][4] // block number 7 from chain B
				gs.addScheduledChange(chainBBlock7, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainCBlock7 := headers[2][0] // block number 7 from chain C
				gs.addScheduledChange(chainCBlock7, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader:                 [2]int{1, 1}, // finalize block number 6 from chain A
			expectedScheduledChangeRootsLen: 1,
			expectedChange: &pendingChange{
				delay: 0,
				nextAuthorities: func() []types.Authority {
					auths, _ := types.GrandpaAuthoritiesRawToAuthorities(
						[]types.GrandpaAuthoritiesRaw{
							{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
							{Key: keyring.KeyFerdie.Public().(*sr25519.PublicKey).AsBytes()},
							{Key: keyring.KeyGeorge.Public().(*sr25519.PublicKey).AsBytes()},
						},
					)
					return auths
				}(),
			},
		},
		"finalized_header_with_no_scheduled_change_should_purge_other_pending_changes": {
			generateForks:                   genericForks,
			expectedScheduledChangeRootsLen: 1,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock8 := headers[0][7] // block 8 from chain A should keep
				gs.addScheduledChange(chainABlock8, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock9 := headers[1][6] // block 9 from chain B should be pruned
				gs.addScheduledChange(chainBBlock9, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainCBlock8 := headers[2][1] // block 8 from chain C should be pruned
				gs.addScheduledChange(chainCBlock8, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader: [2]int{0, 6}, // finalize block number 7 from chain A
		},
		"finalising_header_with_pending_changes_should_return_unfinalized_acestor": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock4 := headers[0][3] // block 4 from chain A
				gs.addScheduledChange(chainABlock4, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				// change on block 5 from chain A should be a child
				//  node of scheduled change on block 4 from chain A
				chainABlock5 := headers[0][5]
				gs.addScheduledChange(chainABlock5, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader: [2]int{0, 6}, // finalize block number 7 from chain A
			wantErr:         fmt.Errorf("failed while applying condition: %w", errUnfinalizedAncestor),
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			blockState := testBlockState(t, db)

			voters := types.NewGrandpaVotersFromAuthorities(genesisAuths)
			gs, err := NewGrandpaStateFromGenesis(db, blockState, voters, nil)
			require.NoError(t, err)

			forks := tt.generateForks(t, gs.blockState)

			if tt.changes != nil {
				tt.changes(gs, forks)
			}

			// saving the current state of scheduled changes to compare
			// with the next state in the case of an error (should keep the same)
			previousScheduledChanges := gs.scheduledChangeRoots

			selectedChain := forks[tt.finalizedHeader[0]]
			selectedHeader := selectedChain[tt.finalizedHeader[1]]

			changeNode, err := gs.scheduledChangeRoots.findApplicable(selectedHeader.Hash(),
				selectedHeader.Number, gs.blockState.IsDescendantOf)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				require.Equal(t, previousScheduledChanges, gs.scheduledChangeRoots)
				return
			}

			if tt.expectedChange != nil {
				require.NoError(t, err)
				require.Equal(t, tt.expectedChange.delay, changeNode.change.delay)
				require.Equal(t, tt.expectedChange.nextAuthorities, changeNode.change.nextAuthorities)
			} else {
				require.Nil(t, changeNode)
			}

			require.Len(t, *gs.scheduledChangeRoots, tt.expectedScheduledChangeRootsLen)
			// make sure all the next scheduled changes are descendant of the finalized hash
			assertDescendantChildren(t,
				selectedHeader.Hash(), gs.blockState.IsDescendantOf, *gs.scheduledChangeRoots)
		})
	}
}

func TestApplyScheduledChange(t *testing.T) {
	t.Parallel()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	genesisGrandpaVoters := []types.GrandpaAuthoritiesRaw{
		{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
		{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
	}

	const sizeOfChain = 10
	genericForks := func(t *testing.T, blockState *BlockState) [][]*types.Header {
		/*
		* create chainA and two forks: chainB and chainC
		*
		*      / -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 (B)
		* 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 (A)
		*                          \ -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 (C)
		 */
		chainA := issueBlocksWithBABEPrimary(t, keyring.KeyAlice, blockState, testGenesisHeader, sizeOfChain)
		chainB := issueBlocksWithBABEPrimary(t, keyring.KeyBob, blockState, chainA[1], sizeOfChain)
		chainC := issueBlocksWithBABEPrimary(t, keyring.KeyCharlie, blockState, chainA[5], sizeOfChain)

		return [][]*types.Header{
			chainA, chainB, chainC,
		}
	}

	tests := map[string]struct {
		finalizedHeader [2]int // 2 index array where the 0 index describes the fork and the 1 index describes the header

		generateForks func(*testing.T, *BlockState) [][]*types.Header
		changes       func(*GrandpaState, [][]*types.Header)

		wantErr                         error
		expectedScheduledChangeRootsLen int
		expectedForcedChangesLen        int
		expectedSetID                   uint64
		expectedAuthoritySet            []types.GrandpaVoter
		changeSetIDAt                   uint
		telemetryMock                   *MockTelemetry
	}{
		"empty_scheduled_changes_only_update_the_forced_changes": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock6 := headers[0][5] // block number 6 from chain A
				gs.addForcedChange(chainABlock6, types.GrandpaForcedChange{
					Delay:              1,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock6 := headers[1][3] // block number 6 from chain B
				gs.addForcedChange(chainBBlock6, types.GrandpaForcedChange{
					Delay:              2,
					BestFinalizedBlock: 3,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyCharlie.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyDave.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader:          [2]int{0, 3},
			expectedForcedChangesLen: 1,
			expectedAuthoritySet: func() []types.GrandpaVoter {
				auths, _ := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
				return types.NewGrandpaVotersFromAuthorities(auths)
			}(),
			telemetryMock: nil,
		},
		"pending_scheduled_changes_should_return_error": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainABlock4 := headers[0][3] // block 4 from chain A
				gs.addScheduledChange(chainABlock4, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				// change on block 5 from chain A should be a child
				// node of scheduled change on block 4 from chain A
				chainABlock5 := headers[0][5]
				gs.addScheduledChange(chainABlock5, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader: [2]int{0, 6}, // finalize block number 7 from chain A
			wantErr: fmt.Errorf(
				"cannot get applicable scheduled change: failed while applying condition: %w", errUnfinalizedAncestor),
			expectedScheduledChangeRootsLen: 1, // expected one root len as the second change is a child
			expectedAuthoritySet: func() []types.GrandpaVoter {
				auths, _ := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
				return types.NewGrandpaVotersFromAuthorities(auths)
			}(),
			telemetryMock: nil,
		},
		"no_changes_to_apply_should_only_update_the_scheduled_roots": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainBBlock6 := headers[1][3] // block 6 from chain B
				gs.addScheduledChange(chainBBlock6, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock8 := headers[1][5] // block number 8 from chain B
				gs.addScheduledChange(chainBBlock8, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader:                 [2]int{2, 1}, // finalize block number 8 from chain C
			expectedScheduledChangeRootsLen: 0,
			expectedAuthoritySet: func() []types.GrandpaVoter {
				auths, _ := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
				return types.NewGrandpaVotersFromAuthorities(auths)
			}(),
			telemetryMock: nil,
		},
		"apply_scheduled_change_should_change_voters_and_set_id": {
			generateForks: genericForks,
			changes: func(gs *GrandpaState, headers [][]*types.Header) {
				chainBBlock6 := headers[1][3] // block 6 from chain B
				gs.addScheduledChange(chainBBlock6, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})

				chainBBlock8 := headers[1][5] // block number 8 from chain B
				err = gs.addScheduledChange(chainBBlock8, types.GrandpaScheduledChange{
					Delay: 0,
					Auths: []types.GrandpaAuthoritiesRaw{
						{Key: keyring.KeyBob.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
						{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
					},
				})
			},
			finalizedHeader: [2]int{1, 3}, // finalize block number 6 from chain B
			// the child (block number 8 from chain B) should be the next scheduled change root
			expectedScheduledChangeRootsLen: 1,
			expectedSetID:                   1,
			changeSetIDAt:                   6,
			expectedAuthoritySet: func() []types.GrandpaVoter {
				auths, _ := types.GrandpaAuthoritiesRawToAuthorities([]types.GrandpaAuthoritiesRaw{
					{Key: keyring.KeyAlice.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyIan.Public().(*sr25519.PublicKey).AsBytes()},
					{Key: keyring.KeyEve.Public().(*sr25519.PublicKey).AsBytes()},
				})
				return types.NewGrandpaVotersFromAuthorities(auths)
			}(),
			telemetryMock: func() *MockTelemetry {
				ctrl := gomock.NewController(t)

				telemetryMock := NewMockTelemetry(ctrl)
				telemetryMock.EXPECT().SendMessage(
					gomock.Eq(&telemetry.AfgApplyingScheduledAuthoritySetChange{Block: "6"}),
				)

				return telemetryMock
			}(),
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			db := NewInMemoryDB(t)
			blockState := testBlockState(t, db)

			genesisAuths, err := types.GrandpaAuthoritiesRawToAuthorities(genesisGrandpaVoters)
			require.NoError(t, err)

			voters := types.NewGrandpaVotersFromAuthorities(genesisAuths)
			gs, err := NewGrandpaStateFromGenesis(db, blockState, voters, tt.telemetryMock)
			require.NoError(t, err)

			forks := tt.generateForks(t, gs.blockState)

			if tt.changes != nil {
				tt.changes(gs, forks)
			}

			selectedFork := forks[tt.finalizedHeader[0]]
			selectedFinalizedHeader := selectedFork[tt.finalizedHeader[1]]

			err = gs.ApplyScheduledChanges(selectedFinalizedHeader)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)

				// ensure the forced changes and scheduled changes
				// are descendant of the latest finalized header
				forcedChangeSlice := *gs.forcedChanges
				for _, forcedChange := range forcedChangeSlice {
					isDescendant, err := gs.blockState.IsDescendantOf(
						selectedFinalizedHeader.Hash(), forcedChange.announcingHeader.Hash())

					require.NoError(t, err)
					require.True(t, isDescendant)
				}

				assertDescendantChildren(t,
					selectedFinalizedHeader.Hash(), gs.blockState.IsDescendantOf, *gs.scheduledChangeRoots)
			}

			require.Len(t, *gs.forcedChanges, tt.expectedForcedChangesLen)
			require.Len(t, *gs.scheduledChangeRoots, tt.expectedScheduledChangeRootsLen)

			currentSetID, err := gs.GetCurrentSetID()
			require.NoError(t, err)
			require.Equal(t, tt.expectedSetID, currentSetID)

			currentVoters, err := gs.GetAuthorities(currentSetID)
			require.NoError(t, err)
			require.Equal(t, tt.expectedAuthoritySet, currentVoters)

			blockNumber, err := gs.GetSetIDChange(currentSetID)
			require.NoError(t, err)
			require.Equal(t, tt.changeSetIDAt, blockNumber)
		})
	}
}
