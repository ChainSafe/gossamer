package availability_store

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestAvailabilityStore_StoreLoadAvailableData(t *testing.T) {
	basePath := t.TempDir()
	as, err := NewAvailabilityStore(Config{basepath: basePath})
	require.NoError(t, err)

	availabeData := AvailableData{
		PoV: parachaintypes.PoV{BlockData: []byte("blockdata")},
		ValidationData: parachaintypes.PersistedValidationData{
			ParentHead: parachaintypes.HeadData{Data: []byte("parentHead")},
		},
	}
	err = as.StoreAvailableData(common.Hash{0x01}, availabeData)
	require.NoError(t, err)

	got, err := as.LoadAvailableData(common.Hash{0x01})
	require.NoError(t, err)
	require.Equal(t, availabeData, got)

	got, err = as.LoadAvailableData(common.Hash{0x02})
	require.EqualError(t, err, "pebble: not found")
	require.Equal(t, AvailableData{}, got)
}
