package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_table(t *testing.T) {
	t.Parallel()

	db := New()
	table := db.NewTable("x")

	db.Set([]byte{1}, []byte{2})

	table.Set([]byte{1}, []byte{3})
	expectedKeyValues := map[string][]byte{
		"\x01":  {2},
		"x\x01": {3},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)

	value, err := table.Get([]byte{1})
	require.NoError(t, err)
	assert.Equal(t, []byte{3}, value)

	err = table.Delete([]byte{1})
	require.NoError(t, err)
	expectedKeyValues = map[string][]byte{
		"\x01": {2},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)

	batch := table.NewWriteBatch()
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

	err = table.DropAll()
	require.NoError(t, err)
	expectedKeyValues = map[string][]byte{
		"\x01": {2},
	}
	assert.Equal(t, expectedKeyValues, db.keyValues)
}
