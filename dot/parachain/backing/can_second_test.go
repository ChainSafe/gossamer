// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
	gomock "go.uber.org/mock/gomock"
)

func ignoreChanVal(t *testing.T, ch chan bool) {
	t.Helper()
	// ignore received value
	<-ch
}

func TestHandleCanSecondMessage(t *testing.T) {
	hash, err := getDummyCommittedCandidateReceipt(t).ToPlain().Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	msg := CanSecondMessage{
		CandidateParaID:      1,
		CandidateRelayParent: getDummyHash(t, 5),
		CandidateHash:        candidateHash,
		ParentHeadDataHash:   getDummyHash(t, 4),
		ResponseCh:           make(chan bool),
	}

	t.Run("relay_parent_is_unknown", func(t *testing.T) {
		cb := CandidateBacking{}

		go ignoreChanVal(t, msg.ResponseCh)
		err := cb.handleCanSecondMessage(msg)
		require.ErrorIs(t, err, errUnknwnRelayParent)
	})
	t.Run("async_backing_is_disabled", func(t *testing.T) {
		cb := CandidateBacking{
			perRelayParent: map[common.Hash]*perRelayParentState{
				msg.CandidateRelayParent: {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{IsEnabled: false},
				},
			},
		}

		go ignoreChanVal(t, msg.ResponseCh)
		err := cb.handleCanSecondMessage(msg)
		require.ErrorIs(t, err, errProspectiveParachainsModeDisabled)
	})

	t.Run("candidate_can_not_be_seconded", func(t *testing.T) {
		cb := CandidateBacking{
			perRelayParent: map[common.Hash]*perRelayParentState{
				msg.CandidateRelayParent: {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
				},
			},
		}

		go ignoreChanVal(t, msg.ResponseCh)
		err := cb.handleCanSecondMessage(msg)
		require.ErrorIs(t, err, errCandidateNotRecognised)
	})

	t.Run("candidate_recognised_by_at_least_one_fragment_tree", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockImplicitView := NewMockImplicitView(ctrl)

		mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(
			gomock.AssignableToTypeOf(common.Hash{}),
			gomock.AssignableToTypeOf(new(parachaintypes.ParaID)),
		).Return([]common.Hash{msg.CandidateRelayParent})

		subSystemToOverseer := make(chan any)

		cb := CandidateBacking{
			SubSystemToOverseer: subSystemToOverseer,
			perRelayParent: map[common.Hash]*perRelayParentState{
				msg.CandidateRelayParent: {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
				},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				getDummyHash(t, 1): {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
					secondedAtDepth: map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]{
						msg.CandidateParaID: {},
					},
				},
			},
			ImplicitView: mockImplicitView,
		}

		go func(subSystemToOverseer chan any) {
			in := <-subSystemToOverseer
			responseCh := in.(parachaintypes.ProspectiveParachainsMessageGetHypotheticalFrontier).ResponseCh
			responseCh <- parachaintypes.HypotheticalFrontierResponses{
				{
					HypotheticalCandidate: parachaintypes.HypotheticalCandidateIncomplete{
						CandidateHash:      candidateHash,
						CandidateParaID:    1,
						ParentHeadDataHash: getDummyHash(t, 4),
						RelayParent:        getDummyHash(t, 5),
					},
					Memberships: []parachaintypes.FragmentTreeMembership{{
						RelayParent: getDummyHash(t, 5),
						Depths:      []uint{1, 2, 3},
					}},
				},
			}
		}(subSystemToOverseer)

		go ignoreChanVal(t, msg.ResponseCh)
		err := cb.handleCanSecondMessage(msg)
		require.NoError(t, err)
	})
}

func TestSecondingSanityCheck(t *testing.T) {
	hash, err := getDummyCommittedCandidateReceipt(t).ToPlain().Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	hypotheticalCandidate := parachaintypes.HypotheticalCandidateIncomplete{
		CandidateHash:      candidateHash,
		CandidateParaID:    1,
		ParentHeadDataHash: getDummyHash(t, 4),
		RelayParent:        getDummyHash(t, 5),
	}

	t.Run("prospective_parachains_mode_enabled_and_candidate_relay_parent_not_allowed_for_parachain", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockImplicitView := NewMockImplicitView(ctrl)

		mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(
			gomock.AssignableToTypeOf(common.Hash{}),
			gomock.AssignableToTypeOf(new(parachaintypes.ParaID)),
		).Return([]common.Hash{})

		cb := CandidateBacking{
			perRelayParent: map[common.Hash]*perRelayParentState{
				hypotheticalCandidate.RelayParent: {},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				getDummyHash(t, 1): {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
				},
			},
			ImplicitView: mockImplicitView,
		}

		membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)

		require.NoError(t, err)
		require.Empty(t, membership)
	})

	t.Run("prospective_parachains_mode_enabled_and_depth_already_occupied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockImplicitView := NewMockImplicitView(ctrl)

		mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(
			gomock.AssignableToTypeOf(common.Hash{}),
			gomock.AssignableToTypeOf(new(parachaintypes.ParaID)),
		).Return([]common.Hash{hypotheticalCandidate.RelayParent})

		subSystemToOverseer := make(chan any)

		cb := CandidateBacking{
			SubSystemToOverseer: subSystemToOverseer,
			perRelayParent: map[common.Hash]*perRelayParentState{
				hypotheticalCandidate.RelayParent: {},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				getDummyHash(t, 1): {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
					secondedAtDepth: map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]{
						hypotheticalCandidate.CandidateParaID: func() *btree.Map[uint, parachaintypes.CandidateHash] {
							var btm btree.Map[uint, parachaintypes.CandidateHash]
							btm.Set(1, hypotheticalCandidate.CandidateHash)
							return &btm
						}(),
					},
				},
			},
			ImplicitView: mockImplicitView,
		}

		go func(subSystemToOverseer chan any) {
			in := <-subSystemToOverseer
			in.(parachaintypes.ProspectiveParachainsMessageGetHypotheticalFrontier).
				ResponseCh <- parachaintypes.HypotheticalFrontierResponses{
				{
					HypotheticalCandidate: hypotheticalCandidate,
					Memberships: []parachaintypes.FragmentTreeMembership{{
						RelayParent: hypotheticalCandidate.RelayParent,
						Depths:      []uint{1, 2, 3},
					}},
				},
			}
		}(subSystemToOverseer)

		membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)
		require.ErrorIs(t, err, errDepthOccupied)
		require.Empty(t, membership)
	})

	t.Run("prospective_parachains_mode_enabled_and_depth_not_occupied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockImplicitView := NewMockImplicitView(ctrl)

		mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(
			gomock.AssignableToTypeOf(common.Hash{}),
			gomock.AssignableToTypeOf(new(parachaintypes.ParaID)),
		).Return([]common.Hash{hypotheticalCandidate.RelayParent})

		subSystemToOverseer := make(chan any)

		cb := CandidateBacking{
			SubSystemToOverseer: subSystemToOverseer,
			perRelayParent: map[common.Hash]*perRelayParentState{
				hypotheticalCandidate.RelayParent: {},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				getDummyHash(t, 1): {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled:          true,
						MaxCandidateDepth:  4,
						AllowedAncestryLen: 2,
					},
					secondedAtDepth: map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]{
						hypotheticalCandidate.CandidateParaID: {},
					},
				},
			},
			ImplicitView: mockImplicitView,
		}

		go func(subSystemToOverseer chan any) {
			in := <-subSystemToOverseer
			in.(parachaintypes.ProspectiveParachainsMessageGetHypotheticalFrontier).
				ResponseCh <- parachaintypes.HypotheticalFrontierResponses{
				{
					HypotheticalCandidate: hypotheticalCandidate,
					Memberships: []parachaintypes.FragmentTreeMembership{{
						RelayParent: hypotheticalCandidate.RelayParent,
						Depths:      []uint{1, 2, 3},
					}},
				},
			}
		}(subSystemToOverseer)

		membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)
		require.NoError(t, err)
		require.Equal(
			t,
			map[common.Hash][]uint{getDummyHash(t, 1): {1, 2, 3}},
			membership,
		)
	})

	t.Run("prospective_parachains_mode_disabled_and_leaf_is_already_occupied", func(t *testing.T) {
		cb := CandidateBacking{
			perRelayParent: map[common.Hash]*perRelayParentState{
				hypotheticalCandidate.RelayParent: {},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				hypotheticalCandidate.RelayParent: {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled: false,
					},
					secondedAtDepth: map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]{
						hypotheticalCandidate.CandidateParaID: func() *btree.Map[uint, parachaintypes.CandidateHash] {
							var btm btree.Map[uint, parachaintypes.CandidateHash]
							btm.Set(0, hypotheticalCandidate.CandidateHash)
							return &btm
						}(),
					},
				},
			},
		}

		membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)
		require.ErrorIs(t, err, errLeafOccupied)
		require.Empty(t, membership)
	})

	t.Run("prospective_parachains_mode_disabled_and_leaf_is_not_occupied", func(t *testing.T) {
		cb := CandidateBacking{
			perRelayParent: map[common.Hash]*perRelayParentState{
				hypotheticalCandidate.RelayParent: {},
			},
			perLeaf: map[common.Hash]*activeLeafState{
				hypotheticalCandidate.RelayParent: {
					prospectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
						IsEnabled: false,
					},
					secondedAtDepth: map[parachaintypes.ParaID]*btree.Map[uint, parachaintypes.CandidateHash]{
						hypotheticalCandidate.CandidateParaID: {},
					},
				},
			},
		}

		membership, err := cb.secondingSanityCheck(hypotheticalCandidate, true)
		require.NoError(t, err)
		require.Equal(
			t,
			map[common.Hash][]uint{hypotheticalCandidate.RelayParent: {0}},
			membership,
		)
	})
}
