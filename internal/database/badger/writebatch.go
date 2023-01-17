// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	badger "github.com/dgraph-io/badger/v3"
)

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
	key = makePrefixedKey(wb.prefix, key)
	err = wb.badgerWriteBatch.Set(key, value)
	return transformError(err)
}

// Delete deletes the given key prefixed with the table prefix
// from the database.
func (wb *writeBatch) Delete(key []byte) (err error) {
	key = makePrefixedKey(wb.prefix, key)
	err = wb.badgerWriteBatch.Delete(key)
	return transformError(err)
}

// Flush flushes the write batch to the database.
func (wb *writeBatch) Flush() (err error) {
	err = wb.badgerWriteBatch.Flush()
	return transformError(err)
}

// Cancel cancels the write batch.
func (wb *writeBatch) Cancel() {
	wb.badgerWriteBatch.Cancel()
}
