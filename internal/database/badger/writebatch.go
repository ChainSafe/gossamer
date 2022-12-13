package badger

import "github.com/dgraph-io/badger/v3"

// writeBatch uses the badger write batch and prefixes
// all keys with a certain given prefix.
type writeBatch struct {
	prefix           []byte
	badgerWriteBatch *badger.WriteBatch
}

func newWriteBatch(prefix []byte, badgerWriteBatch *badger.WriteBatch) *writeBatch {
	return &writeBatch{
		prefix:           prefix,
		badgerWriteBatch: badgerWriteBatch,
	}
}

// Set sets a value at the given key prefixed with the given prefix.
func (wb *writeBatch) Set(key, value []byte) (err error) {
	key = append(wb.prefix, key...)
	return wb.badgerWriteBatch.Set(key, value)
}

// Delete deletes the given key prefixed with the table prefix
// from the database.
func (wb *writeBatch) Delete(key []byte) (err error) {
	key = append(wb.prefix, key...)
	return wb.badgerWriteBatch.Delete(key)
}

// Flush flushes the write batch to the database.
func (wb *writeBatch) Flush() (err error) {
	return wb.badgerWriteBatch.Flush()
}

// Cancel cancels the write batch.
func (wb *writeBatch) Cancel() {
	wb.badgerWriteBatch.Cancel()
}
