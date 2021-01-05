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
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"

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
}

// InstanceConfig represents a runtime instance configuration
type InstanceConfig struct {
	Storage     Storage
	Keystore    *keystore.GenericKeystore
	LogLvl      log.Lvl
	Role        byte
	NodeStorage NodeStorage
	Network     BasicNetwork
	Transaction TransactionState
}

// StorageChangeOperation represents a storage change operation
type StorageChangeOperation byte

//nolint
const (
	SetOp         StorageChangeOperation = 0
	ClearOp       StorageChangeOperation = 1
	ClearPrefixOp StorageChangeOperation = 2
	AppendOp      StorageChangeOperation = 3
	DeleteChildOp StorageChangeOperation = 4
)

// TransactionStorageChange represents a storage change made after ext_storage_start_transaction is called
type TransactionStorageChange struct {
	Operation  StorageChangeOperation
	Prefix     []byte
	KeyToChild []byte // key to child trie, if applicable
	Key        []byte
	Value      []byte
}

// Context is the context for the wasm interpreter's imported functions
type Context struct {
	Storage     Storage
	Allocator   *FreeingBumpHeapAllocator
	Keystore    *keystore.GenericKeystore
	Validator   bool
	NodeStorage NodeStorage
	Network     BasicNetwork
	Transaction TransactionState
	SigVerifier *SignatureVerifier
	// TransactionStorageChanges is used by ext_storage_start_transaction to keep track of
	// changes made after it's called. The next call to ext_storage_commit_transaction will
	// commit all the changes, or if ext_storage_rollback_transaction is called, the changes
	// will be discarded.
	TransactionStorageChanges []*TransactionStorageChange
}

// Signature ...
type Signature struct {
	PubKey    []byte
	Sign      []byte
	Msg       []byte
	KyeTypeID crypto.KeyType
}

func (sig *Signature) verify() error {
	switch sig.KyeTypeID {
	case crypto.Ed25519Type:
		pubKey, err := ed25519.NewPublicKey(sig.PubKey)
		if err != nil {
			return fmt.Errorf("failed to fetch ed25519 public key: %s", err)
		}
		ok, err := pubKey.Verify(sig.Msg, sig.Sign)
		if err != nil || !ok {
			return fmt.Errorf("failed to verify ed25519 signature: %s", err)
		}
	case crypto.Sr25519Type:
		pubKey, err := sr25519.NewPublicKey(sig.PubKey)
		if err != nil {
			return fmt.Errorf("failed to fetch sr25519 public key: %s", err)
		}
		ok, err := pubKey.Verify(sig.Msg, sig.Sign)
		if err != nil || !ok {
			return fmt.Errorf("failed to verify sr25519 signature: %s", err)
		}
	case crypto.Secp256k1Type:
		ok := secp256k1.VerifySignature(sig.PubKey, sig.Msg, sig.Sign)
		if !ok {
			return fmt.Errorf("failed to verify secp256k1 signature")
		}
	}
	return nil
}

// SignatureVerifier ...
type SignatureVerifier struct {
	batch   []*Signature
	init    bool
	inValid bool
	sync.RWMutex
	closeCh chan struct{}
}

// Init ...
func (sv *SignatureVerifier) Init() {
	sv.Lock()
	defer sv.Unlock()

	sv.init = true
	sv.inValid = false
	sv.batch = make([]*Signature, 0)
	sv.closeCh = make(chan struct{})
}

// IsStarted ...
func (sv *SignatureVerifier) IsStarted() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.init
}

// IsInValid ...
func (sv *SignatureVerifier) IsInValid() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.inValid
}

// InValid ...
func (sv *SignatureVerifier) InValid() {
	sv.RLock()
	defer sv.RUnlock()
	sv.inValid = true
}

// Add ...
func (sv *SignatureVerifier) Add(s *Signature) {
	if sv.IsInValid() {
		return
	}

	sv.Lock()
	defer sv.Unlock()
	sv.batch = append(sv.batch, s)
}

// Start signature verification in batch.
func (sv *SignatureVerifier) Start() {
	sv.Init()
	for {
		select {
		case <-sv.closeCh:
			return
		default:
			if sv.IsEmpty() {
				continue
			}

			sv.Lock()
			sign := sv.batch[0]
			sv.batch = sv.batch[1:len(sv.batch)]
			sv.Unlock()

			err := sign.verify()
			if err != nil {
				log.Error("[ext_crypto_start_batch_verify_version_1] %s", err)
				sv.InValid()
				return
			}
		}
	}
}

// Finish waits till batch is finished. Returns true if all the signatures are valid, Otherwise returns false.
func (sv *SignatureVerifier) Finish() bool {
	for !sv.IsEmpty() && !sv.IsInValid() {
		time.Sleep(100 * time.Millisecond)
	}
	close(sv.closeCh)
	return !sv.IsInValid()
}

// IsEmpty ...
func (sv *SignatureVerifier) IsEmpty() bool {
	sv.RLock()
	defer sv.RUnlock()
	return len(sv.batch) == 0
}

// Version struct
type Version struct {
	Spec_name         []byte
	Impl_name         []byte
	Authoring_version int32
	Spec_version      int32
	Impl_version      int32
}

// VersionAPI struct that holds Runtime Version info and API array
type VersionAPI struct {
	RuntimeVersion *Version
	API            []*API_Item
}

// API_Item struct to hold runtime API Name and Version
type API_Item struct {
	Name []byte
	Ver  int32
}

// Decode to scale decode []byte to VersionAPI struct
func (v *VersionAPI) Decode(in []byte) error {
	// decode runtime version
	_, err := scale.Decode(in, v.RuntimeVersion)
	if err != nil {
		return err
	}

	// 1 + len(Spec_name) + 1 + len(Impl_name) + 12 for  3 int32's - 1 (zero index)
	index := len(v.RuntimeVersion.Spec_name) + len(v.RuntimeVersion.Impl_name) + 14

	// read byte at index for qty of apis
	sd := scale.Decoder{Reader: bytes.NewReader(in[index : index+1])}
	numApis, err := sd.DecodeInteger()
	if err != nil {
		return err
	}
	// put index on first value
	index++
	// load api_item objects
	for i := 0; i < int(numApis); i++ {
		ver, err := scale.Decode(in[index+8+(i*12):index+12+(i*12)], int32(0))
		if err != nil {
			return err
		}
		v.API = append(v.API, &API_Item{
			Name: in[index+(i*12) : index+8+(i*12)],
			Ver:  ver.(int32),
		})
	}

	return nil
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
