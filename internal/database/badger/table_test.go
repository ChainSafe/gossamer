// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_table(t *testing.T) {
	settings := Settings{
		Path: t.TempDir(),
	}
	db, err := New(settings)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	// Important:
	// We use a variable instead of a constant for 'prefix'
	// to have a byte slice of length 6 and capacity 8.
	// See the `makePrefixedKey` function in `helpers.go`.
	prefixString := "prefix"
	prefix := []byte(prefixString)
	dbTable := &table{
		prefix:   prefix,
		database: db,
	}

	err = dbTable.Set([]byte{1}, []byte{1})
	require.NoError(t, err)
	assertDBValue(t, db, makePrefixedKey(prefix, []byte{1}), []byte{1})

	err = dbTable.Delete([]byte{1})
	require.NoError(t, err)
	assertDBKeyNotFound(t, db, append(prefix, []byte{1}...))

	err = dbTable.Set([]byte{2}, []byte{2})
	require.NoError(t, err)
	assertDBValue(t, db, append(prefix, []byte{2}...), []byte{2})

	value, err := dbTable.Get([]byte{2})
	require.NoError(t, err)
	assert.Equal(t, []byte{2}, value)

	writeBatch := dbTable.NewWriteBatch()

	err = writeBatch.Set([]byte{3}, []byte{3})
	require.NoError(t, err)
	assertDBKeyNotFound(t, db, append(prefix, []byte{3}...))

	writeBatch.Cancel()

	writeBatch = dbTable.NewWriteBatch()

	err = writeBatch.Set([]byte{1}, []byte{1})
	require.NoError(t, err)

	err = writeBatch.Set([]byte{2}, []byte{2})
	require.NoError(t, err)

	err = writeBatch.Delete([]byte{1})
	require.NoError(t, err)

	err = writeBatch.Flush()
	require.NoError(t, err)

	assertDBKeyNotFound(t, db, makePrefixedKey(prefix, []byte{1}))
	assertDBValue(t, db, makePrefixedKey(prefix, []byte{2}), []byte{2})

}
