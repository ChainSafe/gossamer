// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"testing"

	prospectiveparachains "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestHandleCanSecondMessage(t *testing.T) {
	tests := []struct {
		name             string
		candidateBacking func() *CandidateBacking
		msg              CanSecondMessage
		mockOverseer     func(chan any)
		expectedError    string
		expectedResponse bool
	}{

		{
			name: "unknown_relay_parent",
			candidateBacking: func() *CandidateBacking {
				return &CandidateBacking{
					perRelayParent: make(map[common.Hash]*perRelayParentState),
				}
			},
			msg: CanSecondMessage{
				CandidateRelayParent: common.Hash{0x01},
				ResponseCh:           make(chan bool, 1),
			},
			mockOverseer:     func(overseerChannel chan any) {},
			expectedError:    errUnknwnRelayParent.Error(),
			expectedResponse: false,
		},
		{
			name: "candidate_can_be_seconded",
			candidateBacking: func() *CandidateBacking {
				cb := &CandidateBacking{
					perRelayParent: map[common.Hash]*perRelayParentState{{0x01}: {}},
					ImplicitView:   implicitViewForCanSecondMsg(t),
				}
				return cb
			},
			msg: CanSecondMessage{
				CandidateRelayParent: common.Hash{0x01},
				CandidateHash:        parachaintypes.CandidateHash{Value: common.Hash{0x02}},
				CandidateParaID:      3,
				ParentHeadDataHash:   common.Hash{0x04},
				ResponseCh:           make(chan bool, 1),
			},
			mockOverseer: func(overseerChannel chan any) {
				msg, ok := <-overseerChannel
				require.True(t, ok)

				getMembership, ok := msg.(prospectiveparachains.GetHypotheticalMembership)
				require.True(t, ok)

				getMembership.Response <- []prospectiveparachains.HypotheticalMembershipResponseItem{{
					HypotheticalCandidate: parachaintypes.HypotheticalCandidateIncomplete{
						ClaimedCandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
					},
					HypotheticalMembership: []common.Hash{{0x01}},
				}}
			},
			expectedError:    "",
			expectedResponse: true,
		},
		{
			name: "candidate_cannot_be_seconded",
			candidateBacking: func() *CandidateBacking {
				cb := &CandidateBacking{
					perRelayParent: make(map[common.Hash]*perRelayParentState),
					ImplicitView:   implicitViewForCanSecondMsg(t),
				}
				cb.perRelayParent[common.Hash{0x01}] = &perRelayParentState{}
				return cb
			},
			msg: CanSecondMessage{
				CandidateRelayParent: common.Hash{0x01},
				CandidateHash:        parachaintypes.CandidateHash{Value: common.Hash{0x02}},
				CandidateParaID:      3,
				ParentHeadDataHash:   common.Hash{0x04},
				ResponseCh:           make(chan bool, 1),
			},
			mockOverseer: func(overseerChannel chan any) {
				msg, ok := <-overseerChannel
				require.True(t, ok)

				getMembership, ok := msg.(prospectiveparachains.GetHypotheticalMembership)
				require.True(t, ok)

				getMembership.Response <- []prospectiveparachains.HypotheticalMembershipResponseItem{}
			},
			expectedError:    "",
			expectedResponse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			overseerChannel := make(chan any)
			go tt.mockOverseer(overseerChannel)

			cb := tt.candidateBacking()
			cb.SubSystemToOverseer = overseerChannel

			err := cb.handleCanSecondMessage(tt.msg)
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.expectedError)
			}

			response := <-tt.msg.ResponseCh
			require.Equal(t, tt.expectedResponse, response)
		})
	}
}

// implicitViewForCanSecondMsg returns a mock ImplicitView for the CanSecondMessage test
func implicitViewForCanSecondMsg(t *testing.T) *MockImplicitView {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockImplicitView := NewMockImplicitView(ctrl)
	mockImplicitView.EXPECT().Leaves().Return([]common.Hash{{0x01}})
	mockImplicitView.EXPECT().KnownAllowedRelayParentsUnder(
		gomock.AssignableToTypeOf(common.Hash{}), gomock.AssignableToTypeOf(new(parachaintypes.ParaID)),
	).Return([]common.Hash{{0x01}})

	return mockImplicitView
}
