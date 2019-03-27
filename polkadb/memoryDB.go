package polkadb

import (
	"sync"
	"errors"
)

type MemDatabase struct {
	db map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() (*MemDatabase, error) {
	return &MemDatabase{
		db: make(map[string][]byte),
	}, nil
}

func (db *MemDatabase) Put(k []byte, v []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(k)] = v
	return nil
}

func (db *MemDatabase) Has(k []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(k)]
	return ok, nil
}

func (db *MemDatabase) Get(k []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if v, ok := db.db[string(k)]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	
	delete(db.db, string(key))
	return nil
}