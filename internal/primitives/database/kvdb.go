// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"encoding/binary"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/kvdb"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / A wrapper around [kvdb.KeyValueDB] that implements [Database] interface
type DBAdapter[H runtime.Hash] struct {
	db kvdb.KeyValueDB
}

func NewDBAdapter[H runtime.Hash](db kvdb.KeyValueDB) *DBAdapter[H] {
	return &DBAdapter[H]{
		db: db,
	}
}

func (dba *DBAdapter[H]) readCounter(col ColumnID, key []byte) (counterKey []byte, counter *uint32, err error) {
	// Add a key suffix for the counter
	counterKey = key
	counterKey = append(counterKey, 0)
	val, err := dba.db.Get(uint32(col), counterKey)
	if err != nil {
		return nil, nil, err
	}
	if val != nil {
		if len(val) != 4 {
			return nil, nil, fmt.Errorf("unexpected counter len: %d", len(val))
		}
		counterData := val
		counter := binary.LittleEndian.Uint32(counterData)
		return counterKey, &counter, nil
	}
	return counterKey, nil, nil
}

func (dba *DBAdapter[H]) Commit(transaction Transaction[H]) error {
	tx := kvdb.NewDBTransaction()
	for _, change := range transaction {
		switch change := change.(type) {
		case Set:
			tx.Put(uint32(change.ColumnID), change.Key, change.Value)
		case Remove:
			tx.Delete(uint32(change.ColumnID), change.Key)
		case Store[H]:
			counterKey, counter, err := dba.readCounter(change.ColumnID, change.Hash.Bytes())
			if err != nil {
				return err
			}
			if counter != nil {
				*counter += 1
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, *counter)
				tx.Put(uint32(change.ColumnID), counterKey, buf)
			} else {
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, 1)
				tx.Put(uint32(change.ColumnID), counterKey, buf)
				tx.Put(uint32(change.ColumnID), change.Hash.Bytes(), change.Value)
			}
		case Reference[H]:
			counterKey, counter, err := dba.readCounter(change.ColumnID, change.Hash.Bytes())
			if err != nil {
				return err
			}
			if counter != nil {
				*counter += 1
				buf := make([]byte, 4)
				binary.LittleEndian.PutUint32(buf, *counter)
				tx.Put(uint32(change.ColumnID), counterKey, buf)
			}
		case Release[H]:
			counterKey, counter, err := dba.readCounter(change.ColumnID, change.Hash.Bytes())
			if err != nil {
				return err
			}
			if counter != nil {
				*counter -= 1
				if *counter == 0 {
					tx.Delete(uint32(change.ColumnID), counterKey)
					tx.Delete(uint32(change.ColumnID), change.Hash.Bytes())
				} else {
					buf := make([]byte, 4)
					binary.LittleEndian.PutUint32(buf, *counter)
					tx.Put(uint32(change.ColumnID), counterKey, buf)
				}
			}
		default:
			panic("unreachable")
		}
	}
	return dba.db.Write(tx)
}

func (dba *DBAdapter[H]) Get(col ColumnID, key []byte) []byte {
	val, err := dba.db.Get(uint32(col), key)
	if err != nil {
		panic(err)
	}
	return val
}

func (dba *DBAdapter[H]) Contains(col ColumnID, key []byte) bool {
	has, err := dba.db.HasKey(uint32(col), key)
	if err != nil {
		panic(err)
	}
	return has
}
