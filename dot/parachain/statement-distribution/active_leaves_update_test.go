package statementdistribution

import (
	"sync"
	"testing"

	prospectiveparachainsmessages "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	keystore "github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestHandleActiveLeavesUpdate_HappyPath tests the happy path for handleActiveLeavesUpdate.
func TestHandleActiveLeavesUpdate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	leafHash := common.MustBlake2bHash([]byte("leaf1"))
	activatedLeaf := &parachaintypes.ActivatedLeaf{Hash: leafHash}

	implicitViewMock := NewMockImplicitView(ctrl)
	implicitViewMock.EXPECT().
		ActivateLeaf(leafHash, gomock.Any()).
		Return(nil)
	implicitViewMock.EXPECT().
		AllAllowedRelayParents().
		Return([]common.Hash{leafHash})

	rtInstanceMock := NewMockInstance(ctrl)
	rtInstanceMock.EXPECT().
		ParachainHostDisabledValidators().
		Return([]parachaintypes.ValidatorIndex{}, nil)

	// as the returned session index does not exists
	// in the perSession map, it will be created by handleActiveLeafUpdate
	// the next mocks are needed to ensure the creation of the session state
	rtInstanceMock.EXPECT().
		ParachainHostSessionIndexForChild().
		Return(parachaintypes.SessionIndex(1), nil)

	dummyKeystore := keystore.NewGenericKeystore("generic_test_keystore")
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	err = dummyKeystore.Insert(kp)
	require.NoError(t, err)

	dummyPubKey := parachaintypes.ValidatorID(dummyKeystore.Sr25519PublicKeys()[0].Encode())

	sessionInfoDummy := parachaintypes.SessionInfo{
		ActiveValidatorIndices:  []parachaintypes.ValidatorIndex{0, 1},
		RandomSeed:              [32]byte{},
		DisputePeriod:           parachaintypes.SessionIndex(10),
		Validators:              []parachaintypes.ValidatorID{dummyPubKey, {2}},
		DiscoveryKeys:           []parachaintypes.AuthorityDiscoveryID{{3}, {4}},
		AssignmentKeys:          []parachaintypes.AssignmentID{{5}, {6}},
		ValidatorGroups:         [][]parachaintypes.ValidatorIndex{{0, 1}},
		NCores:                  2,
		ZerothDelayTrancheWidth: 1,
		RelayVRFModuloSamples:   1,
		NDelayTranches:          1,
		NoShowSlots:             1,
		NeededApprovals:         1,
	}

	rtInstanceMock.EXPECT().
		ParachainHostSessionInfo(parachaintypes.SessionIndex(1)).
		Return(&sessionInfoDummy, nil)

	rtInstanceMock.EXPECT().
		ParachainHostMinimumBackingVotes().
		Return(uint32(3), nil)

	featuresBitVec, err := parachaintypes.NewBitVec([]bool{true, true, false, true})
	require.NoError(t, err)

	rtInstanceMock.EXPECT().
		ParachainHostNodeFeatures().
		Return(featuresBitVec, nil)

	// the next runtime instances are needed to create the
	// per relay parent state
	dummyValidatorGroups := &parachaintypes.ValidatorGroups{
		Validators: [][]parachaintypes.ValidatorIndex{{1}, {2}},
		GroupRotationInfo: parachaintypes.GroupRotationInfo{
			SessionStartBlock:      parachaintypes.BlockNumber(100),
			GroupRotationFrequency: parachaintypes.BlockNumber(10),
			Now:                    parachaintypes.BlockNumber(105),
		},
	}
	rtInstanceMock.EXPECT().
		ParachainHostValidatorGroups().
		Return(dummyValidatorGroups, nil)

	dummyClaimQueue := parachaintypes.ClaimQueue{
		parachaintypes.CoreIndex{Index: 0}: {parachaintypes.ParaID(1), parachaintypes.ParaID(2)},
		parachaintypes.CoreIndex{Index: 1}: {parachaintypes.ParaID(3), parachaintypes.ParaID(4)},
	}
	transposedDummyClaimQueue := dummyClaimQueue.ToTransposed()

	rtInstanceMock.EXPECT().
		ParachainHostClaimQueue().
		Return(dummyClaimQueue, nil)

	blockStateMock := NewMockblockState(ctrl)
	blockStateMock.EXPECT().
		GetRuntime(leafHash).
		Return(rtInstanceMock, nil)

	candidatesMock := NewMockcandidatesTracker(ctrl)
	candidatesMock.EXPECT().
		frontierHypotheticals(nil, nil).
		Return([]parachaintypes.HypotheticalCandidate{})

	state := &v2State{
		implicitView:   implicitViewMock,
		perRelayParent: make(map[common.Hash]*perRelayParentState),
		perSession:     make(map[parachaintypes.SessionIndex]*perSessionState),
		peers:          map[string]peerState{},
		keystore:       dummyKeystore,
		candidates:     candidatesMock, // No candidates tracker needed for this test
	}

	overseerCh := make(chan any, 1)
	sd := &StatementDistribution{
		state:               state,
		blockState:          blockStateMock,
		SubSystemToOverseer: overseerCh,
	}

	// start a goroutine to handle the overseer subsystem
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		hypotheticalMsg := <-overseerCh
		msg, ok := hypotheticalMsg.(prospectiveparachainsmessages.GetHypotheticalMembership)
		require.True(t, ok)

		// just return an empty slice
		msg.Response <- []*prospectiveparachainsmessages.HypotheticalMembershipResponseItem{}
	}()

	// The actual sendPeerMessagesForRelayParent will run, but we can't assert its call directly.
	// Instead, we just ensure no panic and no error.
	err = sd.handleActiveLeavesUpdate(activatedLeaf)
	require.NoError(t, err)
	wg.Wait()

	// assertions
	expectedPerRelayParentState := &perRelayParentState{
		localValidator: &localValidatorState{
			gridTracker: newGridTracker(),
			active: &activeValidatorState{
				index:          parachaintypes.ValidatorIndex(0),
				groupIndex:     parachaintypes.GroupIndex(0),
				assignments:    []parachaintypes.ParaID{parachaintypes.ParaID(1), parachaintypes.ParaID(2)},
				clusterTracker: nil, // TODO: use cluster tracker implementation (#4713)
			},
		},
		statementStore:       nil,
		session:              parachaintypes.SessionIndex(1),
		transposedClaimQueue: transposedDummyClaimQueue,
		groupsPerPara: map[parachaintypes.ParaID][]parachaintypes.GroupIndex{
			parachaintypes.ParaID(1): {parachaintypes.GroupIndex(0)},
			parachaintypes.ParaID(2): {parachaintypes.GroupIndex(0)},
			parachaintypes.ParaID(3): {parachaintypes.GroupIndex(1)},
			parachaintypes.ParaID(4): {parachaintypes.GroupIndex(1)},
		},
		disabledValidators: make(map[parachaintypes.ValidatorIndex]struct{}),
		assignmentsPerGroup: map[parachaintypes.GroupIndex][]parachaintypes.ParaID{
			parachaintypes.GroupIndex(0): {parachaintypes.ParaID(1), parachaintypes.ParaID(2)},
			parachaintypes.GroupIndex(1): {parachaintypes.ParaID(3), parachaintypes.ParaID(4)},
		},
	}
	require.Len(t, state.perRelayParent, 1)
	require.Equal(t, expectedPerRelayParentState, state.perRelayParent[leafHash])

	expectedSessionState := newPerSessionState(
		&sessionInfoDummy,
		dummyKeystore,
		3,
		true,
	)
	require.Len(t, state.perSession, 1)
	require.Equal(t, expectedSessionState, state.perSession[parachaintypes.SessionIndex(1)])
}
