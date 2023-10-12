package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestNewCustomParticipationOutcome(t *testing.T) {
	t.Parallel()

	// with
	validOutcome, err := NewCustomParticipationOutcomeVDT(ParticipationOutcomeValid)
	require.NoError(t, err)

	invalidOutcome, err := NewCustomParticipationOutcomeVDT(ParticipationOutcomeInvalid)
	require.NoError(t, err)

	unavailableOutcome, err := NewCustomParticipationOutcomeVDT(ParticipationOutcomeUnAvailable)
	require.NoError(t, err)

	errorOutcome, err := NewCustomParticipationOutcomeVDT(ParticipationOutcomeError)
	require.NoError(t, err)

	// then
	outcome, err := validOutcome.Value()
	require.NoError(t, err)
	require.Equal(t, ValidOutcome{}, outcome)

	outcome, err = invalidOutcome.Value()
	require.NoError(t, err)
	require.Equal(t, InvalidOutcome{}, outcome)

	outcome, err = unavailableOutcome.Value()
	require.NoError(t, err)
	require.Equal(t, UnAvailableOutcome{}, outcome)

	outcome, err = errorOutcome.Value()
	require.NoError(t, err)
	require.Equal(t, ErrorOutcome{}, outcome)
}

func TestParticipationOutcome_Codec(t *testing.T) {
	t.Parallel()

	// with
	outcome, err := NewParticipationOutcomeVDT()
	require.NoError(t, err)
	outcomeList := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(outcome))

	err = outcomeList.Add(ValidOutcome{},
		InvalidOutcome{},
		UnAvailableOutcome{},
		ErrorOutcome{})
	require.NoError(t, err)

	// when
	encoded, err := scale.Marshal(outcomeList)
	require.NoError(t, err)

	decoded := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(outcome))
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// then
	require.Equal(t, outcomeList, decoded)
}
