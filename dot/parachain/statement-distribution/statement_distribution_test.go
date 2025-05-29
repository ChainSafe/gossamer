// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendBackingFreshStatements(t *testing.T) {
	t.Run("should_send_3_statements_to_backing", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		relayParent := common.Hash{0x01}
		groupIndex := parachaintypes.GroupIndex(0)

		confirmedCandidate := &confirmedCandidate{}
		h, err := confirmedCandidate.receipt.Hash()
		require.NoError(t, err)

		candidateHash := parachaintypes.CandidateHash{Value: h}

		validators := []parachaintypes.ValidatorIndex{0, 1, 2}
		initGroups := [][]parachaintypes.ValidatorIndex{
			validators,
			{3, 4, 5},
		}
		sessionState := &perSessionState{
			groups: newGroups(initGroups, 1),
		}

		stmtStore := NewMockstatementStore(ctrl)
		stmtStore.EXPECT().
			freshStatementsForBacking(validators, candidateHash).
			Return([]parachaintypes.SignedStatement{
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Payload:        parachaintypes.NewCompactSeconded(candidateHash),
						Signature:      parachaintypes.ValidatorSignature{0x01, 0x02, 0x03},
					},
				),
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(1),
						Payload:        parachaintypes.NewCompactValid(candidateHash),
						Signature:      parachaintypes.ValidatorSignature{0x04, 0x04, 0x04},
					},
				),
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(2),
						Payload:        parachaintypes.NewCompactValid(candidateHash),
						Signature:      parachaintypes.ValidatorSignature{0x05, 0x05, 0x05},
					},
				),
			})

		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(0),
				parachaintypes.NewCompactSeconded(candidateHash))
		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(1),
				parachaintypes.NewCompactValid(candidateHash))
		stmtStore.EXPECT().
			noteKnownByBacking(parachaintypes.ValidatorIndex(2),
				parachaintypes.NewCompactValid(candidateHash))

		relayParentState := &perRelayParentState{
			statementStore: stmtStore,
		}

		// the channel cap should be the same as the amount of messages
		// exepected to receive from the subsystem otherwise the test
		// will block
		overseerCh := make(chan any, 3)
		sd := &StatementDistribution{
			SubSystemToOverseer: overseerCh,
		}
		err = sd.sendBackingFreshStatements(
			candidateHash,
			groupIndex,
			relayParent,
			relayParentState,
			confirmedCandidate,
			sessionState,
		)
		require.NoError(t, err)

		// create the expected messages that backing should receive
		stmtSeconded := parachaintypes.NewStatementVDT()
		err = stmtSeconded.SetValue(parachaintypes.Seconded(confirmedCandidate.receipt))
		require.NoError(t, err)

		// create the expected messages that backing should receive
		stmtValid := parachaintypes.NewStatementVDT()
		err = stmtValid.SetValue(parachaintypes.Valid(candidateHash))
		require.NoError(t, err)

		expectedMessages := []backing.StatementMessage{
			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtSeconded,
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Signature:      parachaintypes.ValidatorSignature{0x01, 0x02, 0x03},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},

			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtValid,
						ValidatorIndex: parachaintypes.ValidatorIndex(1),
						Signature:      parachaintypes.ValidatorSignature{0x04, 0x04, 0x04},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},

			{
				RelayParent: relayParent,
				SignedFullStatement: parachaintypes.SignedFullStatementWithPVD{
					SignedFullStatement: parachaintypes.SignedFullStatement{
						Payload:        stmtValid,
						ValidatorIndex: parachaintypes.ValidatorIndex(2),
						Signature:      parachaintypes.ValidatorSignature{0x05, 0x05, 0x05},
					},
					PersistedValidationData: confirmedCandidate.pvd,
				},
			},
		}

		for i := 0; i < len(expectedMessages); i++ {
			msg := <-overseerCh
			require.Equal(t, msg, expectedMessages[i])
		}
	})

	t.Run("should_fail_when_confirmed_candidate_does_not_match", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		relayParent := common.Hash{0x01}
		groupIndex := parachaintypes.GroupIndex(0)

		confirmedCandidate := &confirmedCandidate{}
		h, err := confirmedCandidate.receipt.Hash()
		require.NoError(t, err)

		candidateHash := parachaintypes.CandidateHash{Value: h}
		sessionState := &perSessionState{
			groups: newGroups([][]parachaintypes.ValidatorIndex{{0, 1, 2}}, 1),
		}

		stmtStore := NewMockstatementStore(ctrl)
		// returning a compact statement with a candidate hash
		// that when encoded generates a different encoding from the
		// confirmed candidate we passed to sendBackingFreshStatements
		stmtStore.EXPECT().
			freshStatementsForBacking([]parachaintypes.ValidatorIndex{0, 1, 2}, candidateHash).
			Return([]parachaintypes.SignedStatement{
				parachaintypes.SignedStatement(
					parachaintypes.UncheckedSignedCompactStatement{
						ValidatorIndex: parachaintypes.ValidatorIndex(0),
						Payload: parachaintypes.NewCompactSeconded(
							parachaintypes.CandidateHash{Value: common.Hash{0xab, 0xab}}),
					},
				),
			})

		relayParentState := &perRelayParentState{
			statementStore: stmtStore,
		}

		sd := &StatementDistribution{}
		err = sd.sendBackingFreshStatements(
			candidateHash,
			groupIndex,
			relayParent,
			relayParentState,
			confirmedCandidate,
			sessionState,
		)

		require.ErrorIs(t, err, errEncodedStatementsMismatch)
	})
}
