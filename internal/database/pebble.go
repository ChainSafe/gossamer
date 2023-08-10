// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"errors"
	"fmt"
	"os"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

var logger = log.NewFromGlobal(log.AddContext("internal", "database"))
var _ Database = (*pebbleDB)(nil)

var ErrNotFound = pebble.ErrNotFound

type pebbleDB struct {
	path string
	db   *pebble.DB
}

func NewPebble(path string, inMemory bool) (*pebbleDB, error) {
	opts := &pebble.Options{}
	if inMemory {
		opts = &pebble.Options{FS: vfs.NewMem()}
	} else {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, fmt.Errorf("oppening pebble db: %w", err)
	}

	return &pebbleDB{path, db}, nil
}

func (p *pebbleDB) Path() string {
	return p.path
}

func (p *pebbleDB) Put(key, value []byte) error {
	err := p.db.Set(key, value, &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("writing 0x%x with value 0x%x to database: %w",
			key, value, err)
	}
	return nil
}

func (p *pebbleDB) Get(key []byte) (value []byte, err error) {
	value, closer, err := p.db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("getting 0x%x from database: %w", key, err)
	}

	if err := closer.Close(); err != nil {
		return nil, fmt.Errorf("closing after get: %w", err)
	}

	valueCpy := make([]byte, len(value))
	copy(valueCpy[:], value[:])
	return valueCpy, err
}

func (p *pebbleDB) Has(key []byte) (exists bool, err error) {
	value, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	if err := closer.Close(); err != nil {
		return false, fmt.Errorf("closing after get: %w", err)
	}

	return value != nil, err
}

func (p *pebbleDB) Del(key []byte) error {
	err := p.db.Delete(key, &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("deleting 0x%x from database: %w", key, err)
	}

	return nil
}

func (p *pebbleDB) Close() error {
	return p.db.Close()
}

func (p *pebbleDB) Flush() error {
	err := p.db.Flush()
	if err != nil {
		return fmt.Errorf("flushing database: %w", err)
	}

	return nil
}

func (p *pebbleDB) NewBatch() Batch {
	return &pebbleBatch{
		batch: p.db.NewBatch(),
	}
}

func (p *pebbleDB) NewIterator() Iterator {
	return &pebbleIterator{
		p.db.NewIter(nil),
	}
}

func (p *pebbleDB) NewPrefixIterator(prefix []byte) Iterator {
	keyUpperBound := func(b []byte) []byte {
		end := make([]byte, len(b))
		copy(end, b)

		for i := len(end) - 1; i >= 0; i-- {
			end[i] = end[i] + 1
			if end[i] != 0 {
				return end[:i+1]
			}
		}

		return nil
	}

	prefixIterOptions := &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: keyUpperBound(prefix),
	}

	return &pebbleIterator{
		p.db.NewIter(prefixIterOptions),
	}
}
