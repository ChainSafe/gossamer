// Package databse contains common interfaces and errors for all database implementations.
package database

// WriteBatch is a batch of write operations that can be
// flushed to the database/database table, or canceled.
type WriteBatch interface {
	Set(key, value []byte) error
	Delete(key []byte) error
	Flush() error
	Cancel()
}

// Table is a table derived from the database for a particular
// key prefix.
type Table interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Delete(key []byte) error
	NewWriteBatch() (writeBatch WriteBatch)
	DropAll() error
}
