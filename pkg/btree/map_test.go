package btree

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

type dummy struct {
	Field1 uint32
	Field2 [32]byte
}

// MarshalSCALE is used by scale package for marshalling
func (tr *Map[K, V]) mapped() map[K]V {
	m := btree.Map[K, V](*tr)
	// load into map to be marshalled
	mapped := make(map[K]V)
	m.Scan(func(key K, value V) bool {
		mapped[key] = value
		return true
	})
	return mapped
}

func TestMap_MarshalUnmarshal(t *testing.T) {
	m := NewMap[uint32, dummy](2)
	m.Set(uint32(1), dummy{Field1: 1})
	m.Set(uint32(2), dummy{Field1: 2})
	m.Set(uint32(3), dummy{Field1: 3})

	encoded, err := scale.Marshal(m)
	require.NoError(t, err)
	require.Equal(t, 121, len(encoded))

	d, b := m.Get(uint32(1))
	require.Equal(t, d, dummy{Field1: 1})
	require.True(t, b)
	d, b = m.Get(uint32(2))
	require.Equal(t, d, dummy{Field1: 2})
	require.True(t, b)
	d, b = m.Get(uint32(3))
	require.Equal(t, d, dummy{Field1: 3})
	require.True(t, b)

	expected := m.mapped()

	m = NewMap[uint32, dummy](2)
	err = scale.Unmarshal(encoded, m)
	require.NoError(t, err)
	require.Equal(t, expected, m.mapped())
}
