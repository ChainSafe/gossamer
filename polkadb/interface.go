package polkadb

// PutItem wraps the database write operation supported by regular database.
type PutItem interface {
	Put(key []byte, value []byte) error
}

// Database wraps all database operations. All methods are safe for concurrent use.
type Database interface {
	PutItem
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Del(key []byte) error
	NewBatch() Batch
}

// Batch is a write-only operation
type Batch interface {
	PutItem
	ValueSize() int
	Write() error
	Reset()
	Delete(key []byte) error
}
