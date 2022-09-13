// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package full

import (
	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Logger logs formatted strings at the different log levels.
type Logger interface {
	Debug(s string)
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// ChainDBNewBatcher is the chaindb new batcher interface.
type ChainDBNewBatcher interface {
	NewBatch() chaindb.Batch
}

// JournalDatabase is the chaindb interface for the journal database.
type JournalDatabase interface {
	ChainDBNewBatcher
	Getter
}

// GetterPutter combines the Getter and Putter interfaces.
type GetterPutter interface {
	Getter
	Putter
}

// Getter is the database getter interface.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// PutDeleter combines the Putter and Deleter interfaces.
type PutDeleter interface {
	Putter
	Deleter
}

// Putter puts a key value in the database and returns an error.
type Putter interface {
	Put(key, value []byte) error
}

// Deleter deletes a key and returns an error.
type Deleter interface {
	Del(key []byte) error
}

// BlockState is the block state interface to determine
// if a block is the descendant of another block.
type BlockState interface {
	IsDescendantOf(parent, child common.Hash) (bool, error)
}
