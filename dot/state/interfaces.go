// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/json"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
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

// Runtime interface.
type Runtime interface {
	UpdateRuntimeCode([]byte) error
	Stop()
	NodeStorage() runtime.NodeStorage
	NetworkService() runtime.BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s runtime.Storage)
	GetCodeHash() common.Hash
	Version() (version runtime.Version)
	Metadata() (metadata []byte, err error)
	BabeConfigurer
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error)
	CheckInherents()
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
}

// BabeConfigurer returns the babe configuration of the runtime.
type BabeConfigurer interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
