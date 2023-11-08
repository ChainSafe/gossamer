// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime/offchain"
)

// NodeStorageType type to identify offchain storage type
type NodeStorageType byte

// NodeStorageTypePersistent flag to identify offchain storage as persistent (db)
const NodeStorageTypePersistent NodeStorageType = 1

// NodeStorageTypeLocal flog to identify offchain storage as local (memory)
const NodeStorageTypeLocal NodeStorageType = 2

// NodeStorage struct for storage of runtime offchain worker data
type NodeStorage struct {
	LocalStorage      BasicStorage
	PersistentStorage BasicStorage
	BaseDB            BasicStorage
}

// SetLocal persists a key and value into LOCAL node storage
func (n *NodeStorage) SetLocal(k, v []byte) error {
	return n.LocalStorage.Put(k, v)
}

// GetLocal retrieve a key and value from LOCAL node storage
func (n *NodeStorage) GetLocal(k []byte) ([]byte, error) {
	return n.LocalStorage.Get(k)
}

// SetPersistent persists a key and value into PERSISTENT node storage
func (n *NodeStorage) SetPersistent(k, v []byte) error {
	return n.PersistentStorage.Put(k, v)
}

// GetPersistent retrieve a key and value from PERSISTENT node storage
func (n *NodeStorage) GetPersistent(k []byte) ([]byte, error) {
	return n.PersistentStorage.Get(k)
}

type Allocator interface {
	Allocate(mem Memory, size uint32) (uint32, error)
	Deallocate(mem Memory, ptr uint32) error
}

// Context is the context for the wasm interpreter's imported functions
type Context struct {
	Storage         Storage
	Allocator       Allocator
	Keystore        *keystore.GlobalKeystore
	Validator       bool
	NodeStorage     NodeStorage
	Network         BasicNetwork
	Transaction     TransactionState
	SigVerifier     *crypto.SignatureVerifier
	OffchainHTTPSet *offchain.HTTPSet
	Version         *Version
}
