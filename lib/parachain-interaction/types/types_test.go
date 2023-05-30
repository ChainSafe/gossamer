package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestPersistedValidationData(t *testing.T) {
	expected := []byte{12, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0} //nolint:lll

	pvd := PersistedValidationData{
		ParentHead:             []byte{7, 8, 9},
		RelayParentNumber:      uint32(10),
		RelayParentStorageRoot: common.Hash{},
		MaxPovSize:             uint32(1024),
	}

	actual, err := scale.Marshal(pvd)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	newpvd := PersistedValidationData{}
	err = scale.Unmarshal(actual, &newpvd)
	require.NoError(t, err)
	require.Equal(t, pvd, newpvd)
}

func TestOccupiedCoreAssumption(t *testing.T) {
	included := NewOccupiedCoreAssumption()
	err := included.Set(Included{})
	require.NoError(t, err)
	res, err := scale.Marshal(included)
	require.NoError(t, err)
	require.Equal(t, []byte{0}, res)

	timeout := NewOccupiedCoreAssumption()
	err = timeout.Set(TimedOut{})
	require.NoError(t, err)
	res, err = scale.Marshal(timeout)
	require.NoError(t, err)
	require.Equal(t, []byte{1}, res)

	free := NewOccupiedCoreAssumption()
	err = free.Set(Free{})
	require.NoError(t, err)
	res, err = scale.Marshal(free)
	require.NoError(t, err)
	require.Equal(t, []byte{2}, res)
}
