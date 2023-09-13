// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"io"
	"os"
	"path/filepath"
)

type Reader interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
}

type Writer interface {
	Put(key, value []byte) error
	Del(key []byte) error
	Flush() error
}

// Iterator iterates over key/value pairs in ascending key order.
// Must be released after use.
type Iterator interface {
	Valid() bool
	Next() bool
	Key() []byte
	Value() []byte
	First() bool
	Release()
	SeekGE(key []byte) bool
	io.Closer
}

// Batch is a write-only operation.
type Batch interface {
	io.Closer
	Writer

	ValueSize() int
	Reset()
}

// Database wraps all database operations. All methods are safe for concurrent use.
type Database interface {
	Reader
	Writer
	io.Closer

	Path() string
	NewBatch() Batch
	NewIterator() Iterator
	NewPrefixIterator(prefix []byte) Iterator

	Checkpoint() error
}

type Table interface {
	Reader
	Writer
	Path() string
	NewBatch() Batch
	NewIterator() Iterator
}

const DefaultDatabaseDir = "db"

// LoadDatabase will return an instance of database based on basepath
func LoadDatabase(basepath string, inMemory, checkpoint bool, checkpointPath string) (Database, error) {
	nodeDatabaseDir := filepath.Join(basepath, DefaultDatabaseDir)
	return NewPebble(nodeDatabaseDir, inMemory, checkpoint, checkpointPath)
}

func ClearDatabase(basepath string) error {
	nodeDatabaseDir := filepath.Join(basepath, DefaultDatabaseDir)
	return os.RemoveAll(nodeDatabaseDir)
}
