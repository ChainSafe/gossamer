package types

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestDisputeStatus_Codec(t *testing.T) {
	t.Parallel()
	// with
	status, err := NewDisputeStatus()
	require.NoError(t, err)

	statusList := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(status))
	err = statusList.Add(ActiveStatus{},
		ConcludedForStatus{Since: uint64(time.Now().Unix())},
		ConcludedAgainstStatus{Since: uint64(time.Now().Unix())},
		ConfirmedStatus{})
	require.NoError(t, err)

	// when
	encoded, err := scale.Marshal(statusList)
	require.NoError(t, err)

	decoded := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(status))
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)

	// then
	require.Equal(t, statusList, decoded)
}

func TestDisputeStatus_Confirm(t *testing.T) {
	t.Parallel()

	// with
	status, err := NewDisputeStatus()
	require.NoError(t, err)
	err = status.Set(ActiveStatus{})
	require.NoError(t, err)

	// when
	err = status.Confirm()
	require.NoError(t, err)

	// then
	confirmed, err := status.IsConfirmed()
	require.NoError(t, err)
	require.True(t, confirmed)

	isConfirmedConcluded, err := status.IsConfirmedConcluded()
	require.NoError(t, err)
	require.True(t, isConfirmedConcluded)
}

func TestDisputeStatus_ConcludeFor(t *testing.T) {
	t.Parallel()

	// with
	status, err := NewDisputeStatus()
	require.NoError(t, err)
	concludedAt := uint64(time.Now().Unix())
	err = status.Set(ActiveStatus{})
	require.NoError(t, err)

	// when
	err = status.ConcludeFor(concludedAt)
	require.NoError(t, err)

	// then
	confirmed, err := status.IsConcludedFor()
	require.NoError(t, err)
	require.True(t, confirmed)

	isConfirmedConcluded, err := status.IsConfirmedConcluded()
	require.NoError(t, err)
	require.True(t, isConfirmedConcluded)

	concludedAtFromStatus, err := status.ConcludedAt()
	require.NoError(t, err)
	require.Equal(t, &concludedAt, concludedAtFromStatus)
}

func TestDisputeStatus_ConcludeAgainst(t *testing.T) {
	t.Parallel()

	// with
	status, err := NewDisputeStatus()
	require.NoError(t, err)
	concludedAt := uint64(time.Now().Unix())
	err = status.Set(ActiveStatus{})
	require.NoError(t, err)

	// when
	err = status.ConcludeAgainst(concludedAt)
	require.NoError(t, err)

	// then
	confirmed, err := status.IsConcludedAgainst()
	require.NoError(t, err)
	require.True(t, confirmed)

	isConfirmedConcluded, err := status.IsConfirmedConcluded()
	require.NoError(t, err)
	require.True(t, isConfirmedConcluded)

	concludedAtFromStatus, err := status.ConcludedAt()
	require.NoError(t, err)
	require.Equal(t, &concludedAt, concludedAtFromStatus)
}
