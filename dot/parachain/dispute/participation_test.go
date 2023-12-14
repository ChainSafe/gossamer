package dispute

import (
	"context"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	disputeTypes "github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func dummyCandidateCommitments() parachainTypes.CandidateCommitments {
	return parachainTypes.CandidateCommitments{
		UpwardMessages:            nil,
		HorizontalMessages:        nil,
		NewValidationCode:         nil,
		HeadData:                  parachainTypes.HeadData{},
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             0,
	}
}

func dummyValidationCode() parachainTypes.ValidationCode {
	return parachainTypes.ValidationCode{1, 2, 3}
}

func dummyCollator() parachainTypes.CollatorID {
	return parachainTypes.CollatorID{}
}

func dummyCollatorSignature() parachainTypes.CollatorSignature {
	return parachainTypes.CollatorSignature{}
}

func dummyCandidateDescriptorBadSignature(relayParent common.Hash) parachainTypes.CandidateDescriptor {
	zeros := common.Hash{}
	validationCodeHash, err := dummyValidationCode().Hash()
	if err != nil {
		panic(err)
	}

	return parachainTypes.CandidateDescriptor{
		ParaID:                      0,
		RelayParent:                 relayParent,
		Collator:                    dummyCollator(),
		PersistedValidationDataHash: zeros,
		PovHash:                     zeros,
		ErasureRoot:                 zeros,
		ParaHead:                    zeros,
		ValidationCodeHash:          validationCodeHash,
		Signature:                   dummyCollatorSignature(),
	}
}

func dummyCandidateReceiptBadSignature(
	relayParent common.Hash,
	commitments *common.Hash,
) (parachainTypes.CandidateReceipt, error) {
	var (
		err             error
		commitmentsHash common.Hash
	)
	if commitments == nil {
		commitmentsHash, err = dummyCandidateCommitments().Hash()
		if err != nil {
			return parachainTypes.CandidateReceipt{}, err
		}
	} else {
		commitmentsHash = *commitments
	}

	return parachainTypes.CandidateReceipt{
		Descriptor:      dummyCandidateDescriptorBadSignature(relayParent),
		CommitmentsHash: commitmentsHash,
	}, nil
}

func activateLeaf(
	participation Participation,
	blockNumber parachainTypes.BlockNumber,
	sender chan<- any,
) error {
	encodedBlockNumber, err := scale.Marshal(blockNumber)
	if err != nil {
		return fmt.Errorf("failed to encode block number: %w", err)
	}
	parentHash, err := common.Blake2bHash(encodedBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to hash block number: %w", err)
	}

	blockHeader := types.Header{
		ParentHash:     parentHash,
		Number:         uint(blockNumber),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         scale.VaryingDataTypeSlice{},
	}
	blockHash := blockHeader.Hash()

	update := overseer.ActiveLeavesUpdate{
		Activated: &overseer.ActivatedLeaf{
			Hash:   blockHash,
			Number: uint32(blockNumber),
		},
	}

	participation.ProcessActiveLeavesUpdate(sender, update)
	return nil
}

func participate(participation Participation, overseerChannel chan any) error {
	candidateCommitments := dummyCandidateCommitments()
	commitmentsHash, err := candidateCommitments.Hash()
	if err != nil {
		panic(err)
	}
	return participateWithCommitmentsHash(participation, overseerChannel, commitmentsHash)
}

func participateWithCommitmentsHash(
	participation Participation,
	overseerChannel chan any,
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

	participationData := ParticipationData{
		disputeTypes.ParticipationRequest{
			CandidateHash:    candidateHash,
			CandidateReceipt: candidateReceipt,
			Session:          session,
		},
		ParticipationPriorityBestEffort,
	}

	return participation.Queue(overseerChannel, participationData)
}

func TestParticipationHandler_Queue(t *testing.T) {
	t.Parallel()
	t.Run("should_not_queue_the_same_request_if_the_participation_is_already_running", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(11), mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						response := overseer.RecoveryErrorUnavailable
						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							Error: &response,
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.UnAvailableOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						t.Errorf("unexpected message type: %T", msg)
						return
					}
				}
			}

		}()

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		for i := 0; i < MaxParallelParticipation; i++ {
			err = participate(participationHandler, mockOverseer)
			require.NoError(t, err)
		}

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
	t.Run("requests_get_queued_when_out_of_capacity", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		var wg sync.WaitGroup
		participationTest := func() {
			defer wg.Done()
			participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)
			err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
			require.NoError(t, err)

			err = participate(participationHandler, mockOverseer)
			require.NoError(t, err)

			for i := 0; i < MaxParallelParticipation; i++ {
				err = participateWithCommitmentsHash(participationHandler, mockOverseer, common.Hash{byte(i)})
				require.NoError(t, err)
			}

			for i := 0; i < MaxParallelParticipation+1; i++ {
				err = participationHandler.Clear(mockOverseer, common.Hash{byte(i)})
				require.NoError(t, err)
			}
		}

		overseerHandler := func() {
			defer wg.Done()
			counter := 0
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						response := overseer.RecoveryErrorUnavailable
						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							Error: &response,
						}
					default:
						err := fmt.Errorf("unexpected message type: %T", msg)
						require.NoError(t, err)
					}
				case <-ctx.Done():
					err := fmt.Errorf("timeout")
					require.NoError(t, err)
				}

				counter++
				if counter == 5 {
					break
				}
			}
		}
		receiverHandler := func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			select {
			case msg := <-mockReceiver:
				switch message := msg.(type) {
				case disputeTypes.Message[disputeTypes.ParticipationStatement]:
					outcome, err := message.Data.Outcome.Value()
					require.NoError(t, err)
					switch outcome.(type) {
					case disputeTypes.UnAvailableOutcome:
						return
					default:
						err := fmt.Errorf("unexpected outcome: %T", outcome)
						require.NoError(t, err)
					}
				default:
					err := fmt.Errorf("unexpected message type: %T", msg)
					require.NoError(t, err)
				}
			case <-ctx.Done():
				err := fmt.Errorf("timeout")
				require.NoError(t, err)
			}
		}

		wg.Add(3)
		go overseerHandler()
		go receiverHandler()
		go participationTest()
		wg.Wait()
	})
	t.Run("requests_get_queued_on_no_recent_block", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		waitTx := make(chan bool, 100)
		var wg sync.WaitGroup
		participationTest := func() {
			defer wg.Done()
			participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)

			go func() {
				err := participate(participationHandler, mockOverseer)
				require.NoError(t, err)
			}()

			// We have initiated participation, but we'll block `activeLeaf` so that we can check that
			// the participation is queued in race-free way
			<-waitTx

			err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
			require.NoError(t, err)
		}

		// Responds to messages from the test and verifies its behaviour
		requestHandler := func() {
			defer wg.Done()
			select {
			case msg := <-mockOverseer:
				switch message := msg.(type) {
				case overseer.ChainAPIMessage[overseer.BlockNumber]:
					response := uint32(1)
					message.ResponseChannel <- response
					break
				default:
					panic("unknown message type")
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			select {
			case _ = <-mockOverseer:
				panic("should not receive any messages")
			case <-ctx.Done():
				break
			}

			// No activity so the participation is queued => unblock the test
			waitTx <- true

			counter := 0
			select {
			case msg := <-mockOverseer:
				counter++
				switch message := msg.(type) {
				case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
					response := overseer.RecoveryErrorUnavailable
					message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
						Error: &response,
					}
				case disputeTypes.ParticipationStatement:
					outcome, err := message.Outcome.Value()
					require.NoError(t, err)
					switch outcome.(type) {
					case disputeTypes.UnAvailableOutcome:
						return
					default:
						panic("unexpected outcome")
					}
				default:
					panic("unknown message type")
				}
			}

			if counter == 3 {
				return
			}
		}

		wg.Add(2)
		go requestHandler()
		go participationTest()
		wg.Wait()
	})
	t.Run("cannot_participate_if_cannot_recover_the_available_data", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						response := overseer.RecoveryErrorUnavailable
						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							Error: &response,
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.UnAvailableOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
	t.Run("cannot_participate_if_cannot_recover_validation_code", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						availableData := overseer.AvailableData{
							POV:            []byte{},
							ValidationData: overseer.PersistedValidationData{},
						}

						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							AvailableData: &availableData,
							Error:         nil,
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.ErrorOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(nil, nil).Times(1)

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
	t.Run("cast_invalid_vote_if_available_data_is_invalid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						response := overseer.RecoveryErrorInvalid
						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							Error: &response,
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.InvalidOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
	t.Run("cast_invalid_vote_if_validation_fails_or_is_invalid", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)
		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)

		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.CandidateValidationMessage[overseer.ValidateFromExhaustive]:
						if message.Data.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
							message.ResponseChannel <- overseer.ValidationResult{
								IsValid: false,
								Error:   nil,
							}
						}
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						availableData := overseer.AvailableData{
							POV:            []byte{},
							ValidationData: overseer.PersistedValidationData{},
						}

						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							AvailableData: &availableData,
							Error:         nil,
						}

					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.InvalidOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()

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

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						availableData := overseer.AvailableData{
							POV:            []byte{},
							ValidationData: overseer.PersistedValidationData{},
						}

						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							AvailableData: &availableData,
							Error:         nil,
						}
					case overseer.CandidateValidationMessage[overseer.ValidateFromExhaustive]:
						if message.Data.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
							message.ResponseChannel <- overseer.ValidationResult{
								IsValid: false,
								Error:   nil,
								InvalidResult: &overseer.InvalidValidationResult{
									Reason: "commitments hash mismatch",
								},
							}
						} else {
							panic("unexpected message")
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.InvalidOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()

		mockValidationCode := dummyValidationCode()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(&mockValidationCode, nil).Times(1)

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
	t.Run("cast_vote_if_the_validation_passes", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockOverseer := make(chan any)
		mockReceiver := make(chan any)
		mockRuntime := NewMockRuntimeInstance(ctrl)

		participationHandler := NewParticipation(mockReceiver, mockOverseer, mockRuntime)
		err := activateLeaf(participationHandler, parachainTypes.BlockNumber(10), mockOverseer)
		require.NoError(t, err)

		go func() {
			for {
				select {
				case msg := <-mockOverseer:
					switch message := msg.(type) {
					case overseer.ChainAPIMessage[overseer.BlockNumber]:
						response := uint32(1)
						message.ResponseChannel <- response
					case overseer.AvailabilityRecoveryMessage[overseer.RecoverAvailableData]:
						availableData := overseer.AvailableData{
							POV:            []byte{},
							ValidationData: overseer.PersistedValidationData{},
						}

						message.ResponseChannel <- overseer.AvailabilityRecoveryResponse{
							AvailableData: &availableData,
							Error:         nil,
						}
					case overseer.CandidateValidationMessage[overseer.ValidateFromExhaustive]:
						if message.Data.PvfExecTimeoutKind == overseer.PvfExecTimeoutKindApproval {
							message.ResponseChannel <- overseer.ValidationResult{
								IsValid: true,
								Error:   nil,
								ValidResult: &overseer.ValidValidationResult{
									CandidateCommitments:    parachainTypes.CandidateCommitments{},
									PersistedValidationData: parachainTypes.PersistedValidationData{},
								},
							}
						} else {
							panic("unexpected message")
						}
					case disputeTypes.ParticipationStatement:
						outcome, err := message.Outcome.Value()
						require.NoError(t, err)
						switch outcome.(type) {
						case disputeTypes.ValidOutcome:
							continue
						default:
							panic("unexpected outcome")
						}
					default:
						panic("unknown message type")
					}
				}
			}
		}()

		mockValidationCode := dummyValidationCode()
		mockRuntime.EXPECT().ParachainHostValidationCodeByHash(gomock.Any(), gomock.Any()).
			Return(&mockValidationCode, nil).Times(1)

		err = participate(participationHandler, mockOverseer)
		require.NoError(t, err)

		// sleep for a while to ensure we don't have any further results nor recovery requests
		time.Sleep(5 * time.Second)
	})
}
