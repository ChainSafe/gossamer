// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_table(t *testing.T) {
	t.Parallel()

	db := New()
	dbTable := db.NewTable("x")

	db.Set([]byte{1}, []byte{2})

	dbTable.Set([]byte{1}, []byte{3})
	expectedKeyValues := map[string][]byte{
		"\x01":  {2},
		"x\x01": {3},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)

	value, err := dbTable.Get([]byte{1})
	require.NoError(t, err)
	assert.Equal(t, []byte{3}, value)

	err = dbTable.Delete([]byte{1})
	require.NoError(t, err)
	expectedKeyValues = map[string][]byte{
		"\x01": {2},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)

	batch := dbTable.NewWriteBatch()
	err = batch.Set([]byte{1}, []byte{4})
	require.NoError(t, err)
	err = batch.Set([]byte{2}, []byte{5})
	require.NoError(t, err)
	err = batch.Flush()
	require.NoError(t, err)
	expectedKeyValues = map[string][]byte{
		"\x01":  {2},
		"x\x01": {4},
		"x\x02": {5},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)

	typedDBTable := dbTable.(*table)
	for key := range typedDBTable.database.keyValues {
		if strings.HasPrefix(key, typedDBTable.prefix) {
			delete(typedDBTable.database.keyValues, key)
		}
	}
	expectedKeyValues = map[string][]byte{
		"\x01": {2},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)
}
