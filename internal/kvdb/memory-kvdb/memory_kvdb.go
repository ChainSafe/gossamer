package memorykvdb

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"sync"

	"github.com/ChainSafe/gossamer/internal/kvdb"
	"github.com/tidwall/btree"
)

// / A key-value database fulfilling the `KeyValueDB` trait, living in memory.
// / This is generally intended for tests and is not particularly optimized.
type MemoryKVDB struct {
	columns map[uint32]*btree.Map[string, []byte]
	sync.RWMutex
}

var ErrInvalidColumn = errors.New("no such column family")

// / Create an in-memory database with the given number of columns.
// / Columns will be indexable by 0..numCols
func New(numCols uint32) *MemoryKVDB {
	cols := make(map[uint32]*btree.Map[string, []byte])
	for i := uint32(0); i < numCols; i++ {
		cols[i] = btree.NewMap[string, []byte](0)
	}
	return &MemoryKVDB{
		columns: cols,
	}
}

func (im *MemoryKVDB) Get(col uint32, key []byte) (kvdb.DBValue, error) {
	im.RLock()
	defer im.RUnlock()
	_, ok := im.columns[col]
	if !ok {
		return nil, ErrInvalidColumn
	}
	val, found := im.columns[col].Get(string(key))
	if found {
		return val, nil
	}
	return nil, nil
}

func (im *MemoryKVDB) PrefixGet(col uint32, prefix []byte) (kvdb.DBValue, error) {
	im.RLock()
	defer im.RUnlock()
	_, ok := im.columns[col]
	if !ok {
		return nil, ErrInvalidColumn
	}
	var val kvdb.DBValue
	im.columns[col].Scan(func(key string, value []byte) bool {
		idx := strings.Index(key, string(prefix))
		if idx == 0 {
			val = value
			return false
		}
		return true
	})
	return val, nil
}

func (im *MemoryKVDB) Write(transaction kvdb.DBTransaction) error {
	im.Lock()
	defer im.Unlock()
	for _, op := range transaction.Ops {
		switch op := op.(type) {
		case kvdb.InsertDBOp:
			_, ok := im.columns[op.Col()]
			if ok {
				im.columns[op.Col()].Set(string(op.Key()), op.Value)
			}
		case kvdb.DeleteDBOp:
			_, ok := im.columns[op.Col()]
			if ok {
				im.columns[op.Col()].Delete(string(op.Key()))
			}
		case kvdb.DeletePrefixDBOp:
			_, ok := im.columns[op.Col()]
			if ok {
				if len(op.Prefix) == 0 {
					im.columns[op.Col()].Clear()
				} else {
					var keys []string
					startRange := slices.Clone(op.Prefix)
					endRange := kvdb.EndPrefix(op.Prefix)
					if endRange != nil {
						im.columns[op.Col()].Scan(func(key string, value []byte) bool {
							if strings.Compare(key, string(startRange)) >= 0 && strings.Compare(key, string(endRange)) < 0 {
								keys = append(keys, key)
							}
							if strings.Compare(key, string(endRange)) >= 0 {
								return false
							}
							return true
						})
					} else {
						im.columns[op.Col()].Scan(func(key string, value []byte) bool {
							if strings.Compare(key, string(startRange)) >= 0 {
								keys = append(keys, key)
							}
							return true
						})
					}
					for _, key := range keys {
						im.columns[op.Col()].Delete(key)
					}
				}
			}
		default:
			panic("unreachable")
		}
	}
	return nil
}

func (im *MemoryKVDB) Iter(col uint32) iter.Seq2[kvdb.DBKeyValue, error] {
	return func(yield func(kvdb.DBKeyValue, error) bool) {
		im.RLock()
		defer im.RUnlock()
		_, ok := im.columns[col]
		if !ok {
			yield(kvdb.DBKeyValue{}, ErrInvalidColumn)
			return
		}
		im.columns[col].Scan(func(key string, value []byte) bool {
			return yield(kvdb.DBKeyValue{Key: []byte(key), Value: value}, nil)
		})
	}
}

func (im *MemoryKVDB) PrefixIter(col uint32, prefix []byte) iter.Seq2[kvdb.DBKeyValue, error] {
	return func(yield func(kvdb.DBKeyValue, error) bool) {
		im.RLock()
		defer im.RUnlock()
		_, ok := im.columns[col]
		if !ok {
			yield(kvdb.DBKeyValue{}, ErrInvalidColumn)
			return
		}
		im.columns[col].Scan(func(key string, value []byte) bool {
			idx := strings.Index(key, string(prefix))
			if idx == 0 {
				return yield(kvdb.DBKeyValue{Key: []byte(key), Value: value}, nil)
			}
			return true
		})
	}
}

func (im *MemoryKVDB) HasKey(col uint32, key []byte) (bool, error) {
	val, err := im.Get(col, key)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

func (im *MemoryKVDB) HasPrefix(col uint32, prefix []byte) (bool, error) {
	val, err := im.PrefixGet(col, prefix)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}
