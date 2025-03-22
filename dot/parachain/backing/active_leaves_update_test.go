package backing

import (
	"fmt"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestProcessActiveLeavesUpdateSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description        string
		updateSignal       parachaintypes.ActiveLeavesUpdateSignal
		getCandidateBackig func(ctrl *gomock.Controller) *CandidateBacking
		expectedErr        string
		expectedLenOfPerRP int
	}{
		{
			description: "activated_leaf_is_nil",
			updateSignal: parachaintypes.ActiveLeavesUpdateSignal{
				Activated:   nil,
				Deactivated: []common.Hash{{1}},
			},
			getCandidateBackig: func(ctrl *gomock.Controller) *CandidateBacking {
				mockImplicitView := NewMockImplicitView(ctrl)
				mockImplicitView.EXPECT().deactivateLeaf(common.Hash{1})
				mockImplicitView.EXPECT().AllAllowedRelayParents().Return([]common.Hash{})

				backing := CandidateBacking{
					ImplicitView:   mockImplicitView,
					perRelayParent: map[common.Hash]*perRelayParentState{},
					perCandidate:   map[parachaintypes.CandidateHash]*perCandidateState{},
				}

				return &backing
			},
			expectedErr:        "",
			expectedLenOfPerRP: 0,
		},
		{
			description: "error_fetching_implicit_view",
			updateSignal: parachaintypes.ActiveLeavesUpdateSignal{
				Activated: &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
			},
			getCandidateBackig: func(ctrl *gomock.Controller) *CandidateBacking {
				mockImplicitView := NewMockImplicitView(ctrl)
				mockImplicitView.EXPECT().ActivateLeaf(common.Hash{1}).Return(fmt.Errorf("mock error"))
				mockImplicitView.EXPECT().AllAllowedRelayParents().Return([]common.Hash{{1}})

				backing := CandidateBacking{
					ImplicitView: mockImplicitView,
				}

				return &backing
			},
			expectedErr:        "failed to load implicit view",
			expectedLenOfPerRP: 0,
		},
		{
			description: "clean_up_inactive_relay_parents_and_add_activated_leaf",
			updateSignal: parachaintypes.ActiveLeavesUpdateSignal{
				Activated:   &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
				Deactivated: []common.Hash{{2}},
			},
			getCandidateBackig: func(ctrl *gomock.Controller) *CandidateBacking {
				bv, err := parachaintypes.NewBitVec([]bool{false, true})
				require.NoError(t, err)

				mockImplicitView := NewMockImplicitView(ctrl)

				mockImplicitView.EXPECT().ActivateLeaf(common.Hash{1}).Return(nil)
				mockImplicitView.EXPECT().deactivateLeaf(common.Hash{2})
				mockImplicitView.EXPECT().AllAllowedRelayParents().Return([]common.Hash{{1}})
				mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(common.Hash{1}, nil).Return([]common.Hash{{1}})

				mockBlockState := NewMockBlockState(ctrl)
				mockRuntime := NewMockInstance(ctrl)
				mockBlockState.EXPECT().GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).Return(mockRuntime, nil)
				mockRuntime.EXPECT().ParachainHostSessionIndexForChild().Return(parachaintypes.SessionIndex(1), nil)
				mockRuntime.EXPECT().ParachainHostValidators().Return([]parachaintypes.ValidatorID{{1}, {2}, {3}}, nil)
				mockRuntime.EXPECT().ParachainHostNodeFeatures().Return(bv, nil)
				mockRuntime.EXPECT().ParachainHostSessionExecutorParams(gomock.AssignableToTypeOf(parachaintypes.SessionIndex(1))).
					Return(&parachaintypes.ExecutorParams{}, nil)
				mockRuntime.EXPECT().ParachainHostValidatorGroups().Return(&parachaintypes.ValidatorGroups{}, nil)
				mockRuntime.EXPECT().ParachainHostMinimumBackingVotes().Return(uint32(2), nil)
				mockRuntime.EXPECT().ParachainHostClaimQueue().Return(parachaintypes.ClaimQueue{}, nil)
				mockRuntime.EXPECT().ParachainHostDisabledValidators().Return([]parachaintypes.ValidatorIndex{}, nil)

				backing := CandidateBacking{
					ImplicitView: mockImplicitView,
					perRelayParent: map[common.Hash]*perRelayParentState{
						{2}: {},
						{3}: {},
					},
					perCandidate: map[parachaintypes.CandidateHash]*perCandidateState{
						{Value: common.Hash{2}}: {relayParent: common.Hash{2}},
						{Value: common.Hash{3}}: {relayParent: common.Hash{3}},
					},
					BlockState:      mockBlockState,
					perSessionCache: newPerSessionCache(2),
					Keystore:        keystore.NewBasicKeystore("test", crypto.Sr25519Type),
				}

				return &backing
			},
			expectedErr:        "",
			expectedLenOfPerRP: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			backing := tc.getCandidateBackig(ctrl)

			err := backing.ProcessActiveLeavesUpdateSignal(tc.updateSignal)
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, backing.perRelayParent, tc.expectedLenOfPerRP)
		})
	}
}
