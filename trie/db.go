package trie

import "github.com/chainsafe/go-pre/databases"

type Database struct {
	memDB *databases.MemDB
}

func NewDatabase(memdb *databases.MemDB) *Database {
	return &Database{
		memDB: memdb,
	}
}
