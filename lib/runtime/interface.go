// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
)

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
	InitializeBlockVdt(header *types.HeaderVdt) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	FinalizeBlockVdt() (*types.HeaderVdt, error)
	//ExecuteBlock(block *types.Block) ([]byte, error)
	ExecuteBlockVdt(block *types.BlockVdt) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)

	// TODO: parameters and return values for these are undefined in the spec
	CheckInherents()
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
	ClearChildStorage(keyToChild, key []byte) error
	NextKey([]byte) []byte
	ClearPrefixInChild(keyToChild, prefix []byte) error
	GetChildNextKey(keyToChild, key []byte) ([]byte, error)
	GetChild(keyToChild []byte) (*trie.Trie, error)
	ClearPrefix(prefix []byte) error
	BeginStorageTransaction()
	CommitStorageTransaction()
	RollbackStorageTransaction()
}

// BasicNetwork interface for functions used by runtime network state function
type BasicNetwork interface {
	NetworkState() common.NetworkState
}

// BasicStorage interface for functions used by runtime offchain workers
type BasicStorage interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
}

// TransactionState interface for adding transactions to pool
type TransactionState interface {
	AddToPool(vt *transaction.ValidTransaction) common.Hash
}
