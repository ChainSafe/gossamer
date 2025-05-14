package backing

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	"github.com/ChainSafe/gossamer/dot/parachain/util"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
				ch := make(chan any)
				go func() {
					getMin := (<-ch).(messages.GetMinimumRelayParents)
					getMin.Sender <- []messages.ParaIDBlockNumber{{
						ParaId:      100,
						BlockNumber: 2,
					}}
					close(ch)
				}()

				chain := []common.Hash{{3}, {2}, {1}}
				bs := NewMockBlockState(ctrl)
				gomock.InOrder(
					bs.EXPECT().GetHeader(common.Hash{1}).Return(util.GetBlockHeader(t, chain, common.Hash{1}), nil),
					bs.EXPECT().GetHeader(common.Hash{2}).Return(util.GetBlockHeader(t, chain, common.Hash{2}), nil),
				)

				view := util.NewBackingImplicitView(bs, nil)
				err := view.ActivateLeaf(common.Hash{1}, ch)
				require.NoError(t, err)

				backing := CandidateBacking{
					implicitView: view,
					perRelayParent: map[common.Hash]*perRelayParentState{
						{1}: {},
					},
					perCandidate: map[parachaintypes.CandidateHash]*perCandidateState{},
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
				ch := make(chan any)
				go func() {
					getMin := (<-ch).(messages.GetMinimumRelayParents)
					getMin.Sender <- []messages.ParaIDBlockNumber{{
						ParaId:      100,
						BlockNumber: 2,
					}}
					close(ch)
				}()

				chain := []common.Hash{{3}, {2}, {1}}
				bs := NewMockBlockState(ctrl)
				gomock.InOrder(
					bs.EXPECT().GetHeader(common.Hash{1}).Return(util.GetBlockHeader(t, chain, common.Hash{1}), nil),
					bs.EXPECT().GetHeader(common.Hash{2}).Return(util.GetBlockHeader(t, chain, common.Hash{2}), nil),
				)

				view := util.NewBackingImplicitView(bs, nil)
				err := view.ActivateLeaf(common.Hash{1}, ch)
				require.NoError(t, err)

				backing := CandidateBacking{
					implicitView: view,
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

				subsystemToOverseer := make(chan any)
				go func() {
					getMin := (<-subsystemToOverseer).(messages.GetMinimumRelayParents)
					getMin.Sender <- []messages.ParaIDBlockNumber{{
						ParaId:      100,
						BlockNumber: 2,
					}}

					getMin = (<-subsystemToOverseer).(messages.GetMinimumRelayParents)
					getMin.Sender <- []messages.ParaIDBlockNumber{{
						ParaId:      100,
						BlockNumber: 3,
					}}
					close(subsystemToOverseer)
				}()

				chain := []common.Hash{{3}, {2}, {1}}

				bs := NewMockBlockState(ctrl)
				bs.EXPECT().GetHeader(common.Hash{2}).Return(util.GetBlockHeader(t, chain, common.Hash{2}), nil)
				bs.EXPECT().GetHeader(common.Hash{1}).Return(util.GetBlockHeader(t, chain, common.Hash{1}), nil)

				view := util.NewBackingImplicitView(bs, nil)
				// Activate leaf Hash{2}, which will be deactivated later
				err = view.ActivateLeaf(common.Hash{2}, subsystemToOverseer)
				require.NoError(t, err)

				backing := CandidateBacking{
					SubSystemToOverseer: subsystemToOverseer,
					implicitView:        view,
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
			expectedLenOfPerRP: 2,
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
