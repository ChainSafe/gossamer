package memory

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Database(t *testing.T) {
	t.Parallel()

	db := New()

	err := db.Set([]byte{1}, []byte{2})
	require.NoError(t, err)

	value, err := db.Get([]byte{1})
	require.NoError(t, err)
	assert.Equal(t, []byte{2}, value)

	err = db.Delete([]byte{2})
	require.NoError(t, err)

	err = db.Delete([]byte{1})
	require.NoError(t, err)

	_, err = db.Get([]byte{1})
	require.ErrorIs(t, err, database.ErrKeyNotFound)

	err = db.Set([]byte{1}, []byte{2})
	require.NoError(t, err)

	value, err = db.Get([]byte{1})
	require.NoError(t, err)
	assert.Equal(t, []byte{2}, value)

	err = db.DropAll()
	require.NoError(t, err)

	_, err = db.Get([]byte{1})
	require.ErrorIs(t, err, database.ErrKeyNotFound)

	err = db.Close()
	require.NoError(t, err)

	const panicValue = "database is closed"
	assert.PanicsWithValue(t, panicValue, func() {
		_ = db.Set([]byte{1}, []byte{2})
	})
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
		"key not found": {
			db: &Database{
				keyValues: map[string][]byte{},
			},
			key:        []byte{1},
			errWrapped: database.ErrKeyNotFound,
			errMessage: "key not found: 0x01",
		},
		"key found": {
			db: &Database{
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
		expectedDB *Database
	}{
		"set at new key": {
			db: &Database{
				keyValues: map[string][]byte{},
			},
			key:   []byte{1},
			value: []byte{2},
			expectedDB: &Database{
				keyValues: map[string][]byte{
					"\x01": {2},
				},
			},
		},
		"override value at key": {
			db: &Database{
				keyValues: map[string][]byte{
					"\x01": {1}},
			},
			key:   []byte{1},
			value: []byte{2},
			expectedDB: &Database{
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

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedDB, testCase.db)
		})
	}

	t.Run("value mutation safety", func(t *testing.T) {
		t.Parallel()

		database := &Database{
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
		expectedDB *Database
	}{
		"key not found": {
			db: &Database{
				keyValues: map[string][]byte{},
			},
			key: []byte{1},
			expectedDB: &Database{
				keyValues: map[string][]byte{},
			},
		},
		"key deleted": {
			db: &Database{
				keyValues: map[string][]byte{
					"\x01": {1}},
			},
			key: []byte{1},
			expectedDB: &Database{
				keyValues: map[string][]byte{},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := testCase.db.Delete(testCase.key)

			require.NoError(t, err)
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

	database := &Database{
		keyValues: map[string][]byte{},
	}

	err := database.Close()
	require.NoError(t, err)

	expectedDB := &Database{
		closed: true,
	}
	assert.Equal(t, expectedDB, database)

	assert.PanicsWithValue(t, "database is closed", func() {
		_, _ = database.Get([]byte{1})
	})
}

func Test_Database_DropAll(t *testing.T) {
	t.Parallel()

	database := &Database{
		keyValues: map[string][]byte{
			"\x01": {1},
		},
	}

	err := database.DropAll()
	require.NoError(t, err)

	expectedDB := &Database{
		keyValues: map[string][]byte{},
	}
	assert.Equal(t, expectedDB, database)
}
