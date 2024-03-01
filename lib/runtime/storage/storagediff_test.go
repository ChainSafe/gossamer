package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeSet_InsertGet(t *testing.T) {
	changes := newChangeSet()

	key := "key"
	value := []byte("value")

	changes.upsert(key, value)
	val, deleted := changes.get(key)
	require.False(t, deleted)
	require.Equal(t, value, val)
}

func TestChangeSet_InsertDeleteGet(t *testing.T) {
	changes := newChangeSet()

	key := "key"
	value := []byte("value")

	changes.upsert(key, value)
	changes.delete(key)

	val, deleted := changes.get(key)
	require.True(t, deleted)
	require.Nil(t, val)
}
