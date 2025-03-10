package provisioner

import (
	"reflect"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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
	onchainState := parachaintypes.DisputeState{
		ValidatorsFor:     parachaintypes.NewBitVec([]bool{true, false, true, false, true}),
		ValidatorsAgainst: parachaintypes.NewBitVec([]bool{false, true, false, false, true}),
		Start:             1,
		ConcludedAt:       nil,
	}

	localValidKnown := parachaintypes.ValidatorIndex(0)
	localValidUnknown := parachaintypes.ValidatorIndex(3)

	localInvalidKnown := parachaintypes.ValidatorIndex(1)
	localInvalidUnknown := parachaintypes.ValidatorIndex(3)

	require.False(t, IsVoteWorthToKeep(
		localValidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{})},
		),
		onchainState,
	))

	require.True(t, IsVoteWorthToKeep(
		localValidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{})},
		),
		onchainState,
	))

	require.False(t, IsVoteWorthToKeep(
		localInvalidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{})},
		),
		onchainState,
	))

	require.True(t, IsVoteWorthToKeep(
		localInvalidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{})},
		),
		onchainState,
	))

	// double voting - onchain knows
	localDoubleVoteOnchainKnows := parachaintypes.ValidatorIndex(4)
	require.False(t, IsVoteWorthToKeep(
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
	require.True(t, IsVoteWorthToKeep(
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
	require.True(t, IsVoteWorthToKeep(
		localDoubleVoteOnchainDoesntKnow,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		emptyOnchainState,
	))
}
