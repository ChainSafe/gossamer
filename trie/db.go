package trie

//import "sync"

// Database is a high-level wrapper around a db
// TODO: turn this into an actual database interface
type Database struct {
	memDB *MemDB
}

// NewDatabase creates a new Database from a MemDB
func NewDatabase(memdb *MemDB) *Database {
	return &Database{
		memDB: memdb,
	}
}

// MemDB is an in-memory key value store
type MemDB struct {
	db map[string][]byte
	//lock sync.RWMutex
}

// NewMemDatabase returns a new in-memory key value store
func NewMemDatabase() *MemDB {
	return &MemDB{
		db: make(map[string][]byte),
	}
}
