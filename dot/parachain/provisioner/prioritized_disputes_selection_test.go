package provisioner

import (
	"reflect"
	"testing"
	"time"

	disputemessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

type Enum interface {
	SetValue(any) error
}

func setEnumVariant[E Enum](variant any) E {
	var e E
	t := reflect.TypeOf(e)
	elem := reflect.New(t.Elem())
	e = elem.Interface().(E)

	e.SetValue(variant)
	return e
}

func TestShouldKeepVoteBehaves(t *testing.T) {
	validatorsFor := bitVector(t, []bool{true, false, true, false, true})

	validatorsAgainst := bitVector(t, []bool{false, true, false, false, true})

	onchainState := parachaintypes.DisputeState{
		ValidatorsFor:     validatorsFor,
		ValidatorsAgainst: validatorsAgainst,
		Start:             1,
		ConcludedAt:       nil,
	}

	localValidKnown := parachaintypes.ValidatorIndex(0)
	localValidUnknown := parachaintypes.ValidatorIndex(3)

	localInvalidKnown := parachaintypes.ValidatorIndex(1)
	localInvalidUnknown := parachaintypes.ValidatorIndex(3)

	require.False(t, isVoteWorthToKeep(
		localValidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.True(t, isVoteWorthToKeep(
		localValidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.False(t, isVoteWorthToKeep(
		localInvalidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.True(t, isVoteWorthToKeep(
		localInvalidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// double voting - onchain knows
	localDoubleVoteOnchainKnows := parachaintypes.ValidatorIndex(4)
	require.False(t, isVoteWorthToKeep(
		localDoubleVoteOnchainKnows,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// double voting - onchain doesn't know
	localDoubleVoteOnchainDoesntKnow := parachaintypes.ValidatorIndex(0)
	require.True(t, isVoteWorthToKeep(
		localDoubleVoteOnchainDoesntKnow,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// empty onchain state
	emptyOnchainState := parachaintypes.DisputeState{
		ValidatorsFor:     parachaintypes.BitVec{},
		ValidatorsAgainst: parachaintypes.BitVec{},
		Start:             1,
		ConcludedAt:       nil,
	}
	require.True(t, isVoteWorthToKeep(
		localDoubleVoteOnchainDoesntKnow,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		emptyOnchainState,
	))
}

func TestPartitioningHappyCase(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)
	timeNow := uint64(time.Now().Unix())

	// Create one dispute for each partition
	inactiveUnknownOnchain := disputemessages.RecentDispute{
		SessionIndex:  0,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveUnknownOnchain)

	inactiveUnconcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  1,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveUnconcludedOnchain)
	onchainKey := parachaintypes.DisputeKey{
		SessionIndex:  inactiveUnconcludedOnchain.SessionIndex,
		CandidateHash: inactiveUnconcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	activeUnknownOnchain := disputemessages.RecentDispute{
		SessionIndex:  2,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x03}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeUnknownOnchain)

	activeUnconcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x04}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeUnconcludedOnchain)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  activeUnconcludedOnchain.SessionIndex,
		CandidateHash: activeUnconcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	activeConcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  4,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x05}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeConcludedOnchain)
	onchainConcludeBlockNumber := parachaintypes.BlockNumber(3)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  activeConcludedOnchain.SessionIndex,
		CandidateHash: activeConcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, true, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: &onchainConcludeBlockNumber,
	}

	inactiveConcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  5,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x06}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveConcludedOnchain)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  inactiveConcludedOnchain.SessionIndex,
		CandidateHash: inactiveConcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: &onchainConcludeBlockNumber,
	}

	result := partitionRecentDisputes(input, onchain)

	// Check results
	require.Len(t, result.inactiveUnknownOnchain, 1)
	require.Equal(t, result.inactiveUnknownOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveUnknownOnchain.SessionIndex,
			CandidateHash: inactiveUnknownOnchain.CandidateHash,
		},
	)

	require.Len(t, result.inactiveUnconcludedOnchain, 1)
	require.Equal(t, result.inactiveUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveUnconcludedOnchain.SessionIndex,
			CandidateHash: inactiveUnconcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeUnknownOnchain, 1)
	require.Equal(t, result.activeUnknownOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeUnknownOnchain.SessionIndex,
			CandidateHash: activeUnknownOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeUnconcludedOnchain, 1)
	require.Equal(t, result.activeUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeUnconcludedOnchain.SessionIndex,
			CandidateHash: activeUnconcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeConcludedOnchain, 1)
	require.Equal(t, result.activeConcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeConcludedOnchain.SessionIndex,
			CandidateHash: activeConcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.inactiveConcludedOnchain, 1)
	require.Equal(t, result.inactiveConcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveConcludedOnchain.SessionIndex,
			CandidateHash: inactiveConcludedOnchain.CandidateHash,
		},
	)
}

// This test verifies the double voting behaviour. Currently we don't care if a supermajority is
// achieved with or without the 'help' of a double vote (a validator voting for and against at the
// same time). This makes the test a bit pointless but anyway I'm leaving it here to make this
// decision explicit and have the test code ready in case this behaviour needs to be further tested
// in the future. Link to the PR with the discussions: https://github.com/paritytech/polkadot/pull/5567
func TestPartitioningDoubledOnchainVote(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)

	// Dispute A relies on a 'double onchain vote' to conclude. Validator with index 0 has voted
	// both `for` and `against`. Despite that this dispute should be considered 'can conclude
	// onchain'.
	disputeA := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, disputeA)

	// Dispute B has supermajority + 1 votes, so the doubled onchain vote doesn't affect it. It
	// should be considered as 'can conclude onchain'.
	disputeB := disputemessages.RecentDispute{
		SessionIndex:  4,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, disputeB)

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  disputeA.SessionIndex,
		CandidateHash: disputeA.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{true, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  disputeB.SessionIndex,
		CandidateHash: disputeB.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, true, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{true, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	result := partitionRecentDisputes(input, onchain)

	require.Len(t, result.activeUnconcludedOnchain, 0)
	require.Len(t, result.activeConcludedOnchain, 2)
}

func TestPartitioningDuplicatedDispute(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)

	someDispute := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, someDispute)
	input = append(input, someDispute)

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  someDispute.SessionIndex,
		CandidateHash: someDispute.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	result := partitionRecentDisputes(input, onchain)

	require.Len(t, result.activeUnconcludedOnchain, 1)
	require.Equal(t, result.activeUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  someDispute.SessionIndex,
			CandidateHash: someDispute.CandidateHash,
		},
	)
}

func bitVector(t *testing.T, bits []bool) parachaintypes.BitVec {
	t.Helper()

	bv, err := parachaintypes.NewBitVec(bits)
	require.NoError(t, err)
	return bv
}
