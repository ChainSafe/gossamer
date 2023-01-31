// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/json"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// GetSetDeleter has methods to get, set and delete key values.
type GetSetDeleter interface {
	GetPutter
	Deleter
}

// BlockStateDatabase is the database interface for the block state.
type BlockStateDatabase interface {
	GetSetDeleter
	NewWriteBatcher
}

// GetPutter has methods to get and put key values.
type GetPutter interface {
	Getter
	Setter
}

// GetNewWriteBatcher has methods to get values and create a
// new batch.
type GetNewWriteBatcher interface {
	Getter
	NewWriteBatcher
}

// Getter gets a value corresponding to the given key.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// Setter puts a value at the given key and returns an error.
type Setter interface {
	Set(key []byte, value []byte) error
}

// Deleter deletes a value at the given key and returns an error.
type Deleter interface {
	Delete(key []byte) error
}

// NewWriteBatcher creates a new database batch.
type NewWriteBatcher interface {
	NewWriteBatch() database.WriteBatch
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
	BabeGenerateKeyOwnershipProof(slot uint64, offenderPublicKey [32]byte) (types.OpaqueKeyOwnershipProof, error)
	BabeSubmitReportEquivocationUnsignedExtrinsic(types.BabeEquivocationProof, types.OpaqueKeyOwnershipProof) error
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
	GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
		types.GrandpaOpaqueKeyOwnershipProof, error)
	GrandpaSubmitReportEquivocationUnsignedExtrinsic(
		equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.GrandpaOpaqueKeyOwnershipProof,
	) error
}

// BabeConfigurer returns the babe configuration of the runtime.
type BabeConfigurer interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}
