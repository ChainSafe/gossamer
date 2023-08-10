// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"bytes"
)

type table struct {
	db     Database
	prefix []byte
}

var _ Table = (*table)(nil)

func NewTable(db Database, prefix string) Table {
	return &table{
		db:     db,
		prefix: []byte(prefix),
	}
}

func (t *table) Path() string {
	return string(t.prefix)
}

func (t *table) Get(key []byte) ([]byte, error) {
	tableItemKey := bytes.Join([][]byte{t.prefix, key}, nil)
	return t.db.Get(tableItemKey)
}

func (t *table) Has(key []byte) (bool, error) {
	tableItemKey := bytes.Join([][]byte{t.prefix, key}, nil)
	return t.db.Has(tableItemKey)
}

func (t *table) Put(key, value []byte) error {
	tableItemKey := bytes.Join([][]byte{t.prefix, key}, nil)
	return t.db.Put(tableItemKey, value)
}

func (t *table) Del(key []byte) error {
	tableItemKey := bytes.Join([][]byte{t.prefix, key}, nil)
	return t.db.Del(tableItemKey)
}

func (t *table) Flush() error {
	return t.db.Flush()
}

func (t *table) Close() error {
	return t.db.Close()
}

func (t *table) NewBatch() Batch {
	return &tableBatch{
		batch:  t.db.NewBatch(),
		prefix: t.prefix,
	}
}

func (t *table) NewIterator() Iterator {
	return t.db.NewPrefixIterator(t.prefix)
}
