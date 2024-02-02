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
var _ Database = (*PebbleDB)(nil)

var ErrNotFound = pebble.ErrNotFound

type PebbleDB struct {
	path string
	db   *pebble.DB
}

// NewPebble return an pebble db implementation of Database interface
func NewPebble(path string, inMemory bool) (*PebbleDB, error) {
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

	return &PebbleDB{path, db}, nil
}

func (p *PebbleDB) Path() string {
	return p.path
}

func (p *PebbleDB) Put(key, value []byte) error {
	err := p.db.Set(key, value, &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("writing 0x%x with value 0x%x to database: %w",
			key, value, err)
	}
	return nil
}

func (p *PebbleDB) Get(key []byte) (value []byte, err error) {
	value, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}

	valueCpy := make([]byte, len(value))
	copy(valueCpy, value)

	if err := closer.Close(); err != nil {
		return nil, fmt.Errorf("closing after get: %w", err)
	}

	return valueCpy, err
}

func (p *PebbleDB) Has(key []byte) (exists bool, err error) {
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

func (p *PebbleDB) Del(key []byte) error {
	err := p.db.Delete(key, &pebble.WriteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (p *PebbleDB) Close() error {
	return p.db.Close()
}

func (p *PebbleDB) Flush() error {
	err := p.db.Flush()
	if err != nil {
		return fmt.Errorf("flushing database: %w", err)
	}

	return nil
}

// NewBatch returns an implementation of Batch interface using the
// internal database
func (p *PebbleDB) NewBatch() Batch {
	return &pebbleBatch{
		batch: p.db.NewBatch(),
	}
}

// NewIterator returns an implementation of Iterator interface using the
// internal database
func (p *PebbleDB) NewIterator() (Iterator, error) {
	iter := p.db.NewIter(nil)

	return &pebbleIterator{
		iter,
	}, nil
}

// NewPrefixIterator returns an implementation of Iterator over a specific
// keys that contains the prefix
// more info: https://github.com/ChainSafe/gossamer/pull/3434#discussion_r1291503323
func (p *PebbleDB) NewPrefixIterator(prefix []byte) (Iterator, error) {
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

	iter := p.db.NewIter(prefixIterOptions)

	return &pebbleIterator{
		iter,
	}, nil
}
