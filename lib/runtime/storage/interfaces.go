package storage

// Database is an interface to interact with a key value database.
type Database interface {
	Getter
	Putter
}

// Getter is an interface to get values from a
// key value database.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// Putter is an interface to put key value pairs in a
// key value database.
type Putter interface {
	Put(key, value []byte) (err error)
}
