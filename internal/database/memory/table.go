package memory

import (
	"github.com/ChainSafe/gossamer/internal/database"
)

type table struct {
	prefix   string
	database *Database
}

func newTable(prefix string, database *Database) *table {
	return &table{
		prefix:   prefix,
		database: database,
	}
}

// Get retrieves a value from the database using the given key
// prefixed with the table prefix.
// It returns the wrapped error `database.ErrKeyNotFound` if the
// prefixed key is not found.
func (t *table) Get(key []byte) (value []byte, err error) {
	key = []byte(t.prefix + string(key))
	return t.database.Get(key)
}

// Set sets a value at the given key prefixed with the table prefix
// in the database.
func (t *table) Set(key, value []byte) (err error) {
	key = []byte(t.prefix + string(key))
	return t.database.Set(key, value)
}

// Delete deletes the given key prefixed with the table prefix
// from the database. If the key is not found, no error is returned.
func (t *table) Delete(key []byte) (err error) {
	key = []byte(t.prefix + string(key))
	return t.database.Delete(key)
}

// NewWriteBatch returns a new write batch for the database,
// using the table prefix to prefix all keys.
func (t *table) NewWriteBatch() (writeBatch database.WriteBatch) {
	return newWriteBatch(t.prefix, t.database)
}
