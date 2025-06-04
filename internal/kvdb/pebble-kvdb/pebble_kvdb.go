// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pebblekvdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/kvdb"
)

// A key-value database fulfilling the KeyValueDB interface, backed by pebbleDB.
type PebbleKVDB struct {
	db *database.PebbleDB
}

var ErrInvalidColumn = errors.New("no such column family")

// Create an pebbleDB backed database
func New(db *database.PebbleDB) *PebbleKVDB {
	return &PebbleKVDB{
		db: db,
	}
}

func (p *PebbleKVDB) Get(col uint32, key []byte) (kvdb.DBValue, error) {
	compKey := compositeKey(col, key)

	value, err := p.db.Get(compKey)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}
	return value, nil
}

func (p *PebbleKVDB) PrefixGet(col uint32, prefix []byte) (kvdb.DBValue, error) {
	compositePrefix := compositeKey(col, prefix)

	// Use the existing NewPrefixIterator from PebbleDB
	iter, err := p.db.NewPrefixIterator(compositePrefix)
	if err != nil {
		return nil, err
	}
	defer iter.Release()

	if iter.First() {
		return iter.Value(), nil
	}

	return nil, nil
}

func (p *PebbleKVDB) Write(transaction kvdb.DBTransaction) error {
	batch := p.db.NewBatch()
	defer func() {
		if err := batch.Close(); err != nil {
			panic(err)
		}
	}()

	for _, op := range transaction.Ops {
		switch op := op.(type) {
		case kvdb.InsertDBOp:
			compositeKey := compositeKey(op.Col(), op.Key())
			if err := batch.Put(compositeKey, op.Value); err != nil {
				return fmt.Errorf("batch insert: %w", err)
			}

		case kvdb.DeleteDBOp:
			compositeKey := compositeKey(op.Col(), op.Key())
			if err := batch.Del(compositeKey); err != nil {
				return fmt.Errorf("batch delete: %w", err)
			}

		case kvdb.DeletePrefixDBOp:
			if err := p.deletePrefixInBatch(batch, op.Col(), op.Prefix); err != nil {
				return fmt.Errorf("batch delete prefix: %w", err)
			}

		default:
			panic("unreachable")
		}
	}

	if err := batch.Flush(); err != nil {
		return fmt.Errorf("batch flush: %w", err)
	}

	return nil
}

func (p *PebbleKVDB) deletePrefixInBatch(batch database.Batch, col uint32, prefix []byte) error {
	if len(prefix) == 0 {
		return p.deleteAllInColumn(batch, col)
	}

	compositePrefix := compositeKey(col, prefix)

	iter, err := p.db.NewPrefixIterator(compositePrefix)
	if err != nil {
		return err
	}
	defer iter.Release()

	for iter.First(); iter.Valid(); iter.Next() {
		if err := batch.Del(iter.Key()); err != nil {
			return err
		}
	}

	return nil
}

func (p *PebbleKVDB) deleteAllInColumn(batch database.Batch, col uint32) error {
	colPrefix := getColumnPrefix(col)

	iter, err := p.db.NewPrefixIterator(colPrefix)
	if err != nil {
		return err
	}
	defer iter.Release()

	for iter.First(); iter.Valid(); iter.Next() {
		if err := batch.Del(iter.Key()); err != nil {
			return err
		}
	}

	return nil
}

func (p *PebbleKVDB) Iter(col uint32) iter.Seq2[kvdb.DBKeyValue, error] {
	return func(yield func(kvdb.DBKeyValue, error) bool) {
		colPrefix := getColumnPrefix(col)

		// Use the existing NewPrefixIterator from PebbleDB
		pebbleIter, err := p.db.NewPrefixIterator(colPrefix)
		if err != nil {
			yield(kvdb.DBKeyValue{}, err)
			return
		}
		defer pebbleIter.Release()

		for pebbleIter.First(); pebbleIter.Valid(); pebbleIter.Next() {
			compKey := pebbleIter.Key()
			value := pebbleIter.Value()
			originalKey := getOriginalKey(compKey)

			if !yield(kvdb.DBKeyValue{Key: slices.Clone(originalKey), Value: slices.Clone(value)}, nil) {
				return
			}
		}
	}
}

func (p *PebbleKVDB) PrefixIter(col uint32, prefix []byte) iter.Seq2[kvdb.DBKeyValue, error] {
	return func(yield func(kvdb.DBKeyValue, error) bool) {
		compositePrefix := compositeKey(col, prefix)

		pebbleIter, err := p.db.NewPrefixIterator(compositePrefix)
		if err != nil {
			yield(kvdb.DBKeyValue{}, err)
			return
		}
		defer pebbleIter.Release()

		for pebbleIter.First(); pebbleIter.Valid(); pebbleIter.Next() {

			compositeKey := pebbleIter.Key()
			value := pebbleIter.Value()

			// Extract original key (remove column prefix)
			originalKey := getOriginalKey(compositeKey)

			if !yield(kvdb.DBKeyValue{Key: slices.Clone(originalKey), Value: slices.Clone(value)}, nil) {
				return
			}
		}
	}
}

func (p *PebbleKVDB) HasKey(col uint32, key []byte) (bool, error) {
	compositeKey := compositeKey(col, key)
	return p.db.Has(compositeKey)
}

func (p *PebbleKVDB) HasPrefix(col uint32, prefix []byte) (bool, error) {
	val, err := p.PrefixGet(col, prefix)
	if err != nil {
		return false, err
	}
	return val != nil, nil
}

func (p *PebbleKVDB) Close() error {
	return p.db.Close()
}

func (p *PebbleKVDB) Flush() error {
	return p.db.Flush()
}

func compositeKey(col uint32, key []byte) []byte {
	columnBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(columnBytes, col)
	return append(columnBytes, key...)
}

func getColumnPrefix(col uint32) []byte {
	colBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(colBytes, col)
	return colBytes
}

func getOriginalKey(compositeKey []byte) []byte {
	if len(compositeKey) <= 4 {
		return nil
	}
	return compositeKey[4:]
}
