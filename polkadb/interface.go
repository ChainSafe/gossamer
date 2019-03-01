package polkadb

// PutIt wraps the database write operation supported by both batches and regular databases.
type PutIt interface {
	Put(key []byte, value []byte) error
}

// Database wraps all database operations. All methods are safe for concurrent use.
type Database interface {
	PutIt
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Del(key []byte) error
	Close()
}
