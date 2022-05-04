// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
)

//go:generate mockery --name Instance --structname Instance --case underscore --keeptree

// Instance is the interface a v0.8 runtime instance must implement
type Instance interface {
	UpdateRuntimeCode([]byte) error
	CheckRuntimeVersion([]byte) (Version, error)
	Stop()
	NodeStorage() NodeStorage
	NetworkService() BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s Storage) // used to set the TrieState before a runtime call

	GetCodeHash() common.Hash
	Version() (Version, error)
	Metadata() ([]byte, error)
	BabeConfiguration() (*types.BabeConfiguration, error)
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.TransactionPaymentQueryInfo, error)

	CheckInherents() // TODO: use this in block verification process (#1873)

	// parameters and return values for these are undefined in the spec
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
}

// Storage interface
type Storage interface {
	Set(key []byte, value []byte)
	Get(key []byte) []byte
	Root() (common.Hash, error)
	SetChild(keyToChild []byte, child *trie.Trie) error
	SetChildStorage(keyToChild, key, value []byte) error
	GetChildStorage(keyToChild, key []byte) ([]byte, error)
	Delete(key []byte)
	DeleteChild(keyToChild []byte)
	DeleteChildLimit(keyToChild []byte, limit *[]byte) (uint32, bool, error)
	ClearChildStorage(keyToChild, key []byte) error
	NextKey([]byte) []byte
	ClearPrefixInChild(keyToChild, prefix []byte) error
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	GetChild(keyToChild []byte) (*trie.Trie, error)
	ClearPrefix(prefix []byte) error
	ClearPrefixLimit(prefix []byte, limit uint32) (uint32, bool)
	BeginStorageTransaction()
	CommitStorageTransaction()
	RollbackStorageTransaction()
	LoadCode() []byte
}

// BasicNetwork interface for functions used by runtime network state function
type BasicNetwork interface {
	NetworkState() common.NetworkState
}

// BasicStorage interface for functions used by runtime offchain workers
type BasicStorage interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Del(key []byte) error
}

//go:generate mockery --name TransactionState --structname TransactionState --case underscore --keeptree

// TransactionState interface for adding transactions to pool
type TransactionState interface {
	AddToPool(vt *transaction.ValidTransaction) common.Hash
}
