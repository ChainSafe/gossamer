package databases

import "sync"

type MemDB struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() *MemDB {
	return &MemDB{
		db: make(map[string][]byte),
	}
}
