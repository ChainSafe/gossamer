// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/json"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
)

// GetPutDeleter has methods to get, put and delete key values.
type GetPutDeleter interface {
	GetPutter
	Deleter
}

// BlockStateDatabase is the database interface for the block state.
type BlockStateDatabase interface {
	GetPutDeleter
	Haser
	NewBatcher
}

// GetPutter has methods to get and put key values.
type GetPutter interface {
	Getter
	Putter
}

// GetNewBatcher has methods to get values and create a
// new batch.
type GetNewBatcher interface {
	Getter
	NewBatcher
}

// Getter gets a value corresponding to the given key.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// Putter puts a value at the given key and returns an error.
type Putter interface {
	Put(key []byte, value []byte) error
}

// Deleter deletes a value at the given key and returns an error.
type Deleter interface {
	Del(key []byte) error
}

// Haser checks if a value exists at the given key and returns an error.
type Haser interface {
	Has(key []byte) (has bool, err error)
}

// NewBatcher creates a new database batch.
type NewBatcher interface {
	NewBatch() chaindb.Batch
}

// BabeConfigurer returns the babe configuration of the runtime.
type BabeConfigurer interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
