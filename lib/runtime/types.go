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
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime/offchain"
	log "github.com/ChainSafe/log15"
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

// InstanceConfig represents a runtime instance configuration
type InstanceConfig struct {
	Storage     Storage
	Keystore    *keystore.GlobalKeystore
	LogLvl      log.Lvl
	Role        byte
	NodeStorage NodeStorage
	Network     BasicNetwork
	Transaction TransactionState
	CodeHash    common.Hash
}

// Context is the context for the wasm interpreter's imported functions
type Context struct {
	Storage         Storage
	Allocator       *FreeingBumpHeapAllocator
	Keystore        *keystore.GlobalKeystore
	Validator       bool
	NodeStorage     NodeStorage
	Network         BasicNetwork
	Transaction     TransactionState
	SigVerifier     *SignatureVerifier
	OffchainHTTPSet *offchain.HTTPSet
}

// NewValidateTransactionError returns an error based on a return value from TaggedTransactionQueueValidateTransaction
func NewValidateTransactionError(res []byte) error {
	// confirm we have an error
	if res[0] == 0 {
		return nil
	}

	if res[1] == 0 {
		// transaction is invalid
		return ErrInvalidTransaction
	}

	if res[1] == 1 {
		// transaction validity can't be determined
		return ErrUnknownTransaction
	}

	return ErrCannotValidateTx
}
