package availabilitystore

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestAvailabilityStore_StoreLoadAvailableData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as, err := NewAvailabilityStore(inmemoryDB)
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

func TestAvailabilityStore_StoreChuckData(t *testing.T) {
	inmemoryDB := state.NewInMemoryDB(t)
	as, err := NewAvailabilityStore(inmemoryDB)
	require.NoError(t, err)

	err = as.StoreChunk(common.Hash{0x01}, ErasureChunk{Chunk: []byte("chunk1"), Index: 0, Proof: []byte("proof1")})
	require.NoError(t, err)
}
