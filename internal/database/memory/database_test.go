// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

import (
	"sync/atomic"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func atomicTrue() *atomic.Bool {
	b := new(atomic.Bool)
	b.Store(true)
	return b
}

func Test_Database_Get(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		db         *Database
		key        []byte
		value      []byte
		errWrapped error
		errMessage string
	}{
		"database_closed": {
			db: &Database{
				closed: atomicTrue(),
			},
			errWrapped: database.ErrClosed,
			errMessage: "database closed",
		},
		"key_not_found": {
			db: &Database{
				closed:    new(atomic.Bool),
				keyValues: map[string][]byte{},
			},
			key:        []byte{1},
			errWrapped: database.ErrKeyNotFound,
			errMessage: "key not found: 0x01",
		},
		"key_found": {
			db: &Database{
				closed: new(atomic.Bool),
				keyValues: map[string][]byte{
					"\x01": {2},
				},
			},
			key:   []byte{1},
			value: []byte{2},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			value, err := testCase.db.Get(testCase.key)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.value, value)
		})
	}
}

func Test_Database_Set(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		db         *Database
		key        []byte
		value      []byte
		errWrapped error
		errMessage string
		expectedDB *Database
	}{
		"database_is_closed": {
			db: &Database{
				closed: atomicTrue(),
			},
			errWrapped: database.ErrClosed,
			errMessage: "database closed",
			expectedDB: &Database{
				closed: atomicTrue(),
			},
		},
		"set_at_new_key": {
			db: &Database{
				closed:    new(atomic.Bool),
				keyValues: map[string][]byte{},
			},
			key:   []byte{1},
			value: []byte{2},
			expectedDB: &Database{
				closed: new(atomic.Bool),
				keyValues: map[string][]byte{
					"\x01": {2},
				},
			},
		},
		"override_value_at_key": {
			db: &Database{
				closed: new(atomic.Bool),
				keyValues: map[string][]byte{
					"\x01": {1}},
			},
			key:   []byte{1},
			value: []byte{2},
			expectedDB: &Database{
				closed: new(atomic.Bool),
				keyValues: map[string][]byte{
					"\x01": {2},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.db.Set(testCase.key, testCase.value)

			require.ErrorIs(t, err, testCase.errWrapped)
			if testCase.errWrapped != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.expectedDB, testCase.db)
		})
	}

	t.Run("value mutation safety", func(t *testing.T) {
		t.Parallel()

		database := &Database{
			closed:    new(atomic.Bool),
			keyValues: map[string][]byte{},
		}

		key := []byte{1}
		value := []byte{2}
		err := database.Set(key, value)
		require.NoError(t, err)

		value[0]++
		value, err = database.Get(key)
		require.NoError(t, err)
		assert.Equal(t, []byte{2}, value)
	})
}

func Test_Database_Delete(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		db         *Database
		key        []byte
		errWrapped error
		expectedDB *Database
	}{
		"database_closed": {
			db: &Database{
				closed: atomicTrue(),
			},
			errWrapped: database.ErrClosed,
			expectedDB: &Database{
				closed: atomicTrue(),
			},
		},
		"key_not_found": {
			db: &Database{
				closed:    new(atomic.Bool),
				keyValues: map[string][]byte{},
			},
			key: []byte{1},
			expectedDB: &Database{
				closed:    new(atomic.Bool),
				keyValues: map[string][]byte{},
			},
		},
		"key_deleted": {
			db: &Database{
				closed: new(atomic.Bool),
				keyValues: map[string][]byte{
					"\x01": {1}},
			},
			key: []byte{1},
			expectedDB: &Database{
				closed:    new(atomic.Bool),
				keyValues: map[string][]byte{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.db.Delete(testCase.key)

			require.ErrorIs(t, err, testCase.errWrapped)
			assert.Equal(t, testCase.expectedDB, testCase.db)
		})
	}
}

func Test_Database_NewWriteBatch(t *testing.T) {
	t.Parallel()

	database := &Database{
		keyValues: map[string][]byte{},
	}
	writeBatch := database.NewWriteBatch()

	err := writeBatch.Set([]byte{1}, []byte{2})
	require.NoError(t, err)

	err = writeBatch.Flush()
	require.NoError(t, err)

	expectedDB := &Database{
		keyValues: map[string][]byte{
			"\x01": {2},
		},
	}
	assert.Equal(t, expectedDB, database)
}

func Test_Database_Close(t *testing.T) {
	t.Parallel()

	t.Run("already closed", func(t *testing.T) {
		t.Parallel()

		db := &Database{
			closed: atomicTrue(),
		}
		err := db.Close()
		assert.ErrorIs(t, err, database.ErrClosed)
	})

	t.Run("closing", func(t *testing.T) {
		db := &Database{
			closed:    new(atomic.Bool),
			keyValues: map[string][]byte{},
		}

		err := db.Close()
		require.NoError(t, err)

		expectedDB := &Database{
			closed: atomicTrue(),
		}
		assert.Equal(t, expectedDB, db)

		_, err = db.Get([]byte{1})
		assert.ErrorIs(t, err, database.ErrClosed)
	})
}

func Test_Database_DropAll(t *testing.T) {
	t.Parallel()

	t.Run("database is closed", func(t *testing.T) {
		t.Parallel()

		db := &Database{
			closed: atomicTrue(),
		}
		err := db.DropAll()
		assert.ErrorIs(t, err, database.ErrClosed)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		database := &Database{
			closed: new(atomic.Bool),
			keyValues: map[string][]byte{
				"\x01": {1},
			},
		}

		err := database.DropAll()
		require.NoError(t, err)

		expectedDB := &Database{
			closed:    new(atomic.Bool),
			keyValues: map[string][]byte{},
		}
		assert.Equal(t, expectedDB, database)
	})
}
