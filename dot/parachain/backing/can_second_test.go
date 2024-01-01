package backing

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

/*
func TestHandleCanSecondMessage(t *testing.T) {

	hash, err := getDummyCommittedCandidateReceipt(t).ToPlain().Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	msg := CanSecondMessage{
		CandidateParaID:      1,
		CandidateRelayParent: getDummyHash(t, 5),
		CandidateHash:        candidateHash,
		ParentHeadDataHash:   getDummyHash(t, 4),
		resCh:                make(chan bool),
	}

	// // case 1
	// cb := CandidateBacking{}

	// // case 2
	// cb := CandidateBacking{
	// 	perRelayParent: map[common.Hash]perRelayParentState{
	// 		msg.CandidateRelayParent: {
	// 			ProspectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{IsEnabled: false},
	// 		},
	// 	},
	// }

	// case 3
	cb := CandidateBacking{
		perRelayParent: map[common.Hash]perRelayParentState{
			msg.CandidateRelayParent: {
				ProspectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
					IsEnabled:          true,
					MaxCandidateDepth:  4,
					AllowedAncestryLen: 2,
				},
			},
		},
	}

	go func(ch chan bool) {
		// Send a response to the channel
		<-ch
	}(msg.resCh)

	cb.handleCanSecondMessage(msg)
}
*/

func TestSecondingSanityCheck(t *testing.T) {

	hash, err := getDummyCommittedCandidateReceipt(t).ToPlain().Hash()
	require.NoError(t, err)

	candidateHash := parachaintypes.CandidateHash{Value: hash}

	hypotheticalCandidate := parachaintypes.HCIncomplete{
		CandidateHash:      candidateHash,
		CandidateParaID:    1,
		ParentHeadDataHash: getDummyHash(t, 4),
		RelayParent:        getDummyHash(t, 5),
	}

	// // case 1
	// cb := CandidateBacking{}

	// // case 2
	// ctrl := gomock.NewController(t)
	// mockImplicitView := NewMockImplicitView(ctrl)

	// mockImplicitView.EXPECT().knownAllowedRelayParentsUnder(
	// 	gomock.AssignableToTypeOf(common.Hash{}),
	// 	gomock.AssignableToTypeOf(parachaintypes.ParaID(0)),
	// ).Return([]common.Hash{})

	// cb := CandidateBacking{
	// 	perRelayParent: map[common.Hash]perRelayParentState{
	// 		hypotheticalCandidate.RelayParent: {},
	// 	},
	// 	perLeaf: map[common.Hash]ActiveLeafState{
	// 		getDummyHash(t, 1): {
	// 			ProspectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
	// 				IsEnabled:          true,
	// 				MaxCandidateDepth:  4,
	// 				AllowedAncestryLen: 2,
	// 			},
	// 		},
	// 	},
	// 	implicitView: mockImplicitView,
	// }

	// case 3
	ctrl := gomock.NewController(t)
	mockImplicitView := NewMockImplicitView(ctrl)

	mockImplicitView.EXPECT().knownAllowedRelayParentsUnder(
		gomock.AssignableToTypeOf(common.Hash{}),
		gomock.AssignableToTypeOf(parachaintypes.ParaID(0)),
	).Return([]common.Hash{hypotheticalCandidate.RelayParent})

	SubSystemToOverseer := make(chan any)

	cb := CandidateBacking{
		SubSystemToOverseer: SubSystemToOverseer,
		perRelayParent: map[common.Hash]perRelayParentState{
			hypotheticalCandidate.RelayParent: {},
		},
		perLeaf: map[common.Hash]ActiveLeafState{
			getDummyHash(t, 1): {
				ProspectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
					IsEnabled:          true,
					MaxCandidateDepth:  4,
					AllowedAncestryLen: 2,
				},
			},
		},
		implicitView: mockImplicitView,
	}

	go func(SubSystemToOverseer chan any) {
		in := <-SubSystemToOverseer
		in.(parachaintypes.PPMGetHypotheticalFrontier).Ch <- parachaintypes.HypotheticalFrontierResponse{}
		close(SubSystemToOverseer)
	}(SubSystemToOverseer)

	// // case 4
	// ctrl := gomock.NewController(t)
	// mockImplicitView := NewMockImplicitView(ctrl)

	// mockImplicitView.EXPECT().knownAllowedRelayParentsUnder(
	// 	gomock.AssignableToTypeOf(common.Hash{}),
	// 	gomock.AssignableToTypeOf(parachaintypes.ParaID(0)),
	// ).Return([]common.Hash{hypotheticalCandidate.RelayParent})

	// SubSystemToOverseer := make(chan any)

	// cb := CandidateBacking{
	// 	SubSystemToOverseer: SubSystemToOverseer,
	// 	perRelayParent: map[common.Hash]perRelayParentState{
	// 		hypotheticalCandidate.RelayParent: {},
	// 	},
	// 	perLeaf: map[common.Hash]ActiveLeafState{
	// 		getDummyHash(t, 1): {
	// 			ProspectiveParachainsMode: parachaintypes.ProspectiveParachainsMode{
	// 				IsEnabled:          true,
	// 				MaxCandidateDepth:  4,
	// 				AllowedAncestryLen: 2,
	// 			},
	// 		},
	// 	},
	// 	implicitView: mockImplicitView,
	// }

	// go func(SubSystemToOverseer chan any) {
	// 	in := <-SubSystemToOverseer
	// 	in.(parachaintypes.PPMGetHypotheticalFrontier).Ch <- parachaintypes.HypotheticalFrontierResponse{
	// 		{
	// 			HypotheticalCandidate: hypotheticalCandidate,
	// 			FragmentTreeMembership: []parachaintypes.FragmentTreeMembership{{
	// 				RelayParent: hypotheticalCandidate.RelayParent,
	// 				Depths:      []uint{1, 2, 3},
	// 			}},
	// 		},
	// 	}
	// 	close(SubSystemToOverseer)
	// }(SubSystemToOverseer)

	cb.secondingSanityCheck(hypotheticalCandidate, true)
}
