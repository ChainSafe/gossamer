package database

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type countValue struct {
	Count uint32
	Value []byte
}
type MemDB[H runtime.Hash] struct {
	inner map[ColumnID]map[string]countValue
	sync.RWMutex
}

func NewMemDB[H runtime.Hash]() MemDB[H] {
	return MemDB[H]{
		inner: make(map[ColumnID]map[string]countValue),
	}
}

func (mdb *MemDB[H]) Commit(transaction Transaction[H]) error {
	mdb.Lock()
	defer mdb.Unlock()
	for _, change := range transaction {
		switch change := change.(type) {
		case ChangeSet:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			mdb.inner[change.ColumnID][string(change.Key)] = countValue{1, change.Value}
		case ChangeRemove:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			delete(mdb.inner[change.ColumnID], string(change.Key))
		case ChangeStore[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Key.String()]
			if ok {
				cv.Count += 1
				mdb.inner[change.ColumnID][change.Key.String()] = cv
			} else {
				mdb.inner[change.ColumnID][change.Key.String()] = countValue{1, change.Value}
			}
		case ChangeReference[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Key.String()]
			if ok {
				cv.Count += 1
				mdb.inner[change.ColumnID][change.Key.String()] = cv
			}
		case ChangeRelease[H]:
			_, ok := mdb.inner[change.ColumnID]
			if !ok {
				mdb.inner[change.ColumnID] = make(map[string]countValue)
			}
			cv, ok := mdb.inner[change.ColumnID][change.Key.String()]
			if ok {
				cv.Count -= 1
				if cv.Count == 0 {
					delete(mdb.inner[change.ColumnID], change.Key.String())
				} else {
					mdb.inner[change.ColumnID][change.Key.String()] = cv
				}
			}
		}
	}
	return nil
}

func (mdb *MemDB[H]) Get(col ColumnID, key []byte) *[]byte {
	mdb.RLock()
	defer mdb.RUnlock()
	_, ok := mdb.inner[col]
	if !ok {
		return nil
	}
	cv, ok := mdb.inner[col][string(key)]
	if ok {
		return &cv.Value
	}
	return nil
}
