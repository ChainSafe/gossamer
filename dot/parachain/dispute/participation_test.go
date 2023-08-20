package dispute

import (
	"github.com/ChainSafe/gossamer/dot/parachain"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewParticipation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockRuntime := parachain.NewMockRuntimeInstance(ctrl)
	mockSender := overseer.NewMockSender(ctrl)

	participation := NewParticipation(mockSender, mockRuntime)
	assert.NotNil(t, participation, "should not be nil")
}

func TestParticipationHandler_Queue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueue := NewQueue()
	mockSender := overseer.NewMockSender(ctrl)
	mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

	participationHandler := &ParticipationHandler{
		queue:   mockQueue,
		sender:  mockSender,
		runtime: mockRuntime,
	}

	ctx := overseer.Context{
		Sender: mockSender,
	}

	// Set up the mock behavior
	mockRequest := ParticipationRequest{
		candidateHash: common.Hash{1},
		candidateReceipt: parachainTypes.CandidateReceipt{
			Descriptor: parachainTypes.CandidateDescriptor{
				ParaID:                      1,
				RelayParent:                 common.Hash{},
				Collator:                    parachainTypes.CollatorID{},
				PersistedValidationDataHash: common.Hash{},
				PovHash:                     common.Hash{},
				ErasureRoot:                 common.Hash{},
				Signature:                   parachainTypes.CollatorSignature{},
				ParaHead:                    common.Hash{},
				ValidationCodeHash:          parachainTypes.ValidationCodeHash{},
			},
			CommitmentsHash: common.MustHexToHash("0xa54a8dce5fd2a27e3715f99e4241f674a48f4529f77949a4474f5b283b823535"),
		},
		session: 1,
	}

	mockSender.EXPECT().SendMessage(gomock.Any()).Return(nil)

	// Call the method under test
	err := participationHandler.Queue(ctx, mockRequest, ParticipationPriorityHigh)
	require.NoError(t, err)
}
