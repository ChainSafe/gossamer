// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_writeBatch(t *testing.T) {
	t.Parallel()

	db := &Database{
		keyValues: map[string][]byte{
			"\x03": {2},
		},
	}
	writeBatch := db.NewWriteBatch()

	err := writeBatch.Set([]byte{1}, []byte{2})
	require.NoError(t, err)
	err = writeBatch.Delete([]byte{3})
	require.NoError(t, err)

	writeBatch.Cancel()

	expectedDB := &Database{
		keyValues: map[string][]byte{
			"\x03": {2},
		},
	}
	assert.Equal(t, expectedDB, db)

	err = writeBatch.Set([]byte{1}, []byte{2})
	require.NoError(t, err)
	err = writeBatch.Delete([]byte{3})
	require.NoError(t, err)

	err = writeBatch.Flush()
	require.NoError(t, err)

	expectedDB = &Database{
		keyValues: map[string][]byte{
			"\x01": {2},
		},
	}
	assert.Equal(t, expectedDB, db)
}

func Test_writeBatch_Set(t *testing.T) {
	t.Parallel()

	wb := &writeBatch{}

	key := []byte{1}
	value := []byte{2}

	err := wb.Set(key, value)
	require.NoError(t, err)

	expectedWb := &writeBatch{
		operations: []operation{{
			kind:  operationSet,
			key:   "\x01",
			value: []byte{2},
		}},
	}
	assert.Equal(t, expectedWb, wb)

	// Check it is resistant to value mutation.
	value[0]++
	assert.Equal(t, expectedWb, wb)
}

func Test_writeBatch_Delete(t *testing.T) {
	t.Parallel()

	wb := &writeBatch{}

	key := []byte{1}

	err := wb.Delete(key)
	require.NoError(t, err)

	expectedWb := &writeBatch{
		operations: []operation{{
			kind: operationDelete,
			key:  "\x01",
		}},
	}
	assert.Equal(t, expectedWb, wb)
}

func Test_writeBatch_Flush(t *testing.T) {
	t.Parallel()

	wb := &writeBatch{
		database: &Database{
			keyValues: map[string][]byte{},
		},
		operations: []operation{{
			kind:  operationSet,
			key:   "\x02",
			value: []byte{3},
		}, {
			kind:  operationSet,
			key:   "\x01",
			value: []byte{2},
		}, {
			kind:  operationDelete,
			key:   "\x01",
			value: []byte{2},
		}},
	}

	err := wb.Flush()
	require.NoError(t, err)

	expectedWb := &writeBatch{
		database: &Database{
			keyValues: map[string][]byte{
				"\x02": {3},
			},
		},
	}
	assert.Equal(t, expectedWb, wb)
}

func Test_writeBatch_Cancel(t *testing.T) {
	t.Parallel()

	wb := &writeBatch{
		operations: []operation{{
			kind:  operationSet,
			key:   "\x02",
			value: []byte{3},
		}},
	}

	wb.Cancel()

	expectedWb := &writeBatch{}
	assert.Equal(t, expectedWb, wb)
}
