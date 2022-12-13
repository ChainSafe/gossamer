// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

type operationKind uint8

const (
	operationSet operationKind = iota
	operationDelete
)

type operation struct {
	kind  operationKind
	key   string
	value []byte
}

// writeBatch implements an in-memory write batch and prefixes
// all keys with a certain given prefix.
// It is NOT thread safe, but its Flush operation is thread safe for
// the database injected.
type writeBatch struct {
	prefix     string
	database   *Database
	operations []operation
}

func newWriteBatch(prefix string, database *Database) *writeBatch {
	return &writeBatch{
		prefix:   prefix,
		database: database,
	}
}

// Set sets a value at the given key prefixed with the given prefix.
func (wb *writeBatch) Set(key, value []byte) (err error) {
	op := operation{
		kind:  operationSet,
		key:   wb.prefix + string(key),
		value: copyBytes(value),
	}
	wb.operations = append(wb.operations, op)
	return nil
}

// Delete deletes the given key prefixed with the table prefix
// from the database.
func (wb *writeBatch) Delete(key []byte) (err error) {
	op := operation{
		kind: operationDelete,
		key:  wb.prefix + string(key),
	}
	wb.operations = append(wb.operations, op)
	return nil
}

// Flush flushes the write batch to the database.
func (wb *writeBatch) Flush() (err error) {
	wb.database.mutex.Lock()
	defer wb.database.mutex.Unlock()
	defer wb.Cancel()

	for _, op := range wb.operations {
		switch op.kind {
		case operationSet:
			wb.database.keyValues[op.key] = op.value
		case operationDelete:
			delete(wb.database.keyValues, op.key)
		}
	}

	return nil
}

// Cancel cancels the write batch.
func (wb *writeBatch) Cancel() {
	wb.operations = nil
}
