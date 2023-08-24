package dispute

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func participate(participation Participation, context overseer.Context) error {
	candidateCommitments := dummyCandidateCommitments()
	commitmentsHash, err := candidateCommitments.Hash()
	if err != nil {
		panic(err)
	}
	return participateWithCommitmentsHash(participation, context, commitmentsHash)
}

func participateWithCommitmentsHash(
	participation Participation,
	context overseer.Context,
	commitmentsHash common.Hash,
) error {
	candidateReceipt, err := dummyCandidateReceiptBadSignature(common.Hash{}, &common.Hash{})
	if err != nil {
		return fmt.Errorf("failed to create candidate receipt: %w", err)
	}
	candidateReceipt.CommitmentsHash = commitmentsHash
	session := parachainTypes.SessionIndex(1)

	candidateHash, err := candidateReceipt.Hash()
	if err != nil {
		return fmt.Errorf("failed to hash candidate receipt: %w", err)
	}

	participationRequest := ParticipationRequest{
		candidateHash:    candidateHash,
		candidateReceipt: candidateReceipt,
		session:          session,
	}

	return participation.Queue(context, participationRequest, ParticipationPriorityBestEffort)
}

func TestNewParticipation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockRuntime := parachain.NewMockRuntimeInstance(ctrl)
	mockSender := overseer.NewMockSender(ctrl)

	participation := NewParticipation(mockSender, mockRuntime)
	require.NotNil(t, participation, "should not be nil")
}

func TestParticipationHandler_Queue(t *testing.T) {
	t.Parallel()
	t.Run("should_not_queue_the_same_request_if_the_participation_is_already_running", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		ctx := overseer.Context{
			Sender: mockSender,
		}
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(11))
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				//message := msg.(overseer.ChainAPIMessage)
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.AvailabilityRecoveryMessage:
				//message := msg.(overseer.AvailabilityRecoveryMessage)
				response := overseer.RecoveryErrorUnavailable
				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					Error: &response,
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		}).Times(1)
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.UnAvailableOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})

		err = participate(participationHandler, ctx)
		require.NoError(t, err)

		for i := 0; i < MaxParallelParticipation; i++ {
			err = participate(participationHandler, ctx)
			require.NoError(t, err)
		}

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	t.Run("requests_get_queued_when_out_of_capacity", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		var wg sync.WaitGroup
		participationTest := func() {
			defer wg.Done()
			participationHandler := NewParticipation(mockSender, mockRuntime)
			ctx := overseer.Context{
				Sender: mockSender,
			}
			err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
			require.NoError(t, err)

			err = participate(participationHandler, ctx)
			require.NoError(t, err)

			for i := 0; i < MaxParallelParticipation; i++ {
				err = participateWithCommitmentsHash(participationHandler, ctx, common.Hash{byte(i)})
				require.NoError(t, err)
			}

			for i := 0; i < MaxParallelParticipation+1; i++ {
				err = participationHandler.Clear(common.Hash{byte(i)})
				require.NoError(t, err)
			}
		}

		requestHandler := func() {
			defer wg.Done()
			// sendMessage is called 4 times for the first 3+1 requests
			// sendMessage is called once for getBlockNumber request
			// feed is called 4 times for the requests while sending the results
			mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
				switch message := msg.(type) {
				case overseer.ChainAPIMessage:
					response := uint32(1)
					message.ResponseChannel <- &response
				case overseer.AvailabilityRecoveryMessage:
					response := overseer.RecoveryErrorUnavailable
					message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
						Error: &response,
					}
				default:
					return fmt.Errorf("unknown message type")
				}

				return nil
			}).Times(5)
			mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
				statement := msg.(ParticipationStatement)
				outcome, err := statement.Outcome.Value()
				require.NoError(t, err)
				switch outcome.(type) {
				case types.UnAvailableOutcome:
					return nil
				default:
					panic("invalid outcome")
				}
			}).Times(4)

			time.Sleep(2 * time.Second)
		}

		wg.Add(2)
		go participationTest()
		go requestHandler()
		wg.Wait()
	})

	t.Run("requests_get_queued_on_no_recent_block", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		var wg sync.WaitGroup

		waitTx := make(chan bool, 100)
		participationTest := func() {
			defer wg.Done()

			participationHandler := NewParticipation(mockSender, mockRuntime)
			context := overseer.Context{
				Sender: mockSender,
			}

			go func() {
				err := participate(participationHandler, context)
				require.NoError(t, err)
			}()

			// We have initiated participation, but we'll block `activeLeaf` so that we can check that
			// the participation is queued in race-free way
			<-waitTx

			err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
			require.NoError(t, err)

			time.Sleep(2 * time.Second)
		}

		// Responds to messages from the test and verifies its behaviour
		requestHandler := func() {
			defer wg.Done()

			// If we receive `BlockNumber` request this implicitly proves that the participation is queued
			mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
				switch message := msg.(type) {
				case overseer.ChainAPIMessage:
					response := uint32(1)
					message.ResponseChannel <- &response
				case overseer.AvailabilityRecoveryMessage:
					response := overseer.RecoveryErrorUnavailable
					message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
						Error: &response,
					}
				default:
					return fmt.Errorf("unknown message type")
				}

				return nil
			}).Times(2)
			mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
				statement := msg.(ParticipationStatement)
				outcome, err := statement.Outcome.Value()
				require.NoError(t, err)
				switch outcome.(type) {
				case types.UnAvailableOutcome:
					return nil
				default:
					panic("unexpected outcome")
				}
			})

			time.Sleep(5 * time.Second)
			waitTx <- true
		}

		wg.Add(2)
		go participationTest()
		go requestHandler()
		wg.Wait()
	})

	t.Run("cannot_participate_if_cannot_recover_the_available_data", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		ctx := overseer.Context{
			Sender: mockSender,
		}
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.AvailabilityRecoveryMessage:
				response := overseer.RecoveryErrorUnavailable
				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					Error: &response,
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		}).Times(1)
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.UnAvailableOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})

		err = participate(participationHandler, ctx)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	t.Run("cannot_participate_if_cannot_recover_validation_code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		context := overseer.Context{
			Sender: mockSender,
		}

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.AvailabilityRecoveryMessage:
				availableData := overseer.AvailableData{
					POV:            []byte{},
					ValidationData: overseer.PersistedValidationData{},
				}

				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					AvailableData: &availableData,
					Error:         nil,
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		})
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.ErrorOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(nil, nil).Times(1)

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		err = participate(participationHandler, context)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	t.Run("cast_invalid_vote_if_available_data_is_invalid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		context := overseer.Context{
			Sender: mockSender,
		}

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		err = participate(participationHandler, context)
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.AvailabilityRecoveryMessage:
				response := overseer.RecoveryErrorInvalid
				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					Error: &response,
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		})
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.InvalidOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	t.Run("cast_invalid_vote_if_validation_fails_or_is_invalid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		context := overseer.Context{
			Sender: mockSender,
		}

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		err = participate(participationHandler, context)
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.ValidateFromChainState:
				if message.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
					message.ResponseChannel <- overseer.ValidationResult{
						IsValid: false,
						Error:   nil,
					}
				}
			case overseer.AvailabilityRecoveryMessage:
				availableData := overseer.AvailableData{
					POV:            []byte{},
					ValidationData: overseer.PersistedValidationData{},
				}

				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					AvailableData: &availableData,
					Error:         nil,
				}

			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		}).Times(2)
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.InvalidOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})
		mockValidationCode := dummyValidationCode()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(&mockValidationCode, nil).Times(1)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	// TODO: currently, the candidate validation doesn't support all the error types
	// this test is only setting it as string, but we need to set the appropriate error type
	// refer https://github.com/ChainSafe/gossamer/issues/3426
	t.Run("cast_invalid_vote_if_the_commitments_mismatch", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		ctx := overseer.Context{
			Sender: mockSender,
		}
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.AvailabilityRecoveryMessage:
				availableData := overseer.AvailableData{
					POV:            []byte{},
					ValidationData: overseer.PersistedValidationData{},
				}

				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					AvailableData: &availableData,
					Error:         nil,
				}
			case overseer.ValidateFromChainState:
				if message.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
					message.ResponseChannel <- overseer.ValidationResult{
						IsValid: false,
						Error:   nil,
						InvalidResult: &overseer.InvalidValidationResult{
							Reason: "commitments hash mismatch",
						},
					}
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		}).Times(2)
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.InvalidOutcome:

				return nil
			default:
				panic("unexpected outcome")
			}
		})
		mockValidationCode := dummyValidationCode()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(&mockValidationCode, nil).Times(1)

		err = participate(participationHandler, ctx)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})

	t.Run("cast_vote_if_the_validation_passes", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSender := overseer.NewMockSender(ctrl)
		mockRuntime := parachain.NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockSender, mockRuntime)
		ctx := overseer.Context{
			Sender: mockSender,
		}
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10))
		require.NoError(t, err)

		mockSender.EXPECT().SendMessage(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			switch message := msg.(type) {
			case overseer.ChainAPIMessage:
				response := uint32(1)
				message.ResponseChannel <- &response
			case overseer.AvailabilityRecoveryMessage:
				availableData := overseer.AvailableData{
					POV:            []byte{},
					ValidationData: overseer.PersistedValidationData{},
				}

				message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
					AvailableData: &availableData,
					Error:         nil,
				}
			case overseer.ValidateFromChainState:
				if message.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
					message.ResponseChannel <- overseer.ValidationResult{
						IsValid: true,
						Error:   nil,
						ValidResult: &overseer.ValidValidationResult{
							CandidateCommitments:    parachainTypes.CandidateCommitments{},
							PersistedValidationData: parachainTypes.PersistedValidationData{},
						},
					}
				}
			default:
				return fmt.Errorf("unknown message type")
			}

			return nil
		}).Times(2)
		mockSender.EXPECT().Feed(gomock.Any()).DoAndReturn(func(msg interface{}) error {
			statement := msg.(ParticipationStatement)
			outcome, err := statement.Outcome.Value()
			require.NoError(t, err)
			switch outcome.(type) {
			case types.ValidOutcome:
				return nil
			default:
				panic("unexpected outcome")
			}
		})
		mockValidationCode := dummyValidationCode()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(&mockValidationCode, nil).Times(1)

		err = participate(participationHandler, ctx)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
}
