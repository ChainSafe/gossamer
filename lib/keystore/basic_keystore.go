// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
)

var (
	ErrKeyTypeNotSupported = errors.New("given key type is not supported by this keystore")
)

// BasicKeystore holds keys of a certain type
type BasicKeystore struct {
	name Name
	typ  crypto.KeyType
	keys map[common.Address]KeyPair // map of public key encodings to keypairs
	lock sync.RWMutex
}

// NewBasicKeystore creates a new BasicKeystore with the given key type
func NewBasicKeystore(name Name, typ crypto.KeyType) *BasicKeystore {
	return &BasicKeystore{
		name: name,
		typ:  typ,
		keys: make(map[common.Address]KeyPair),
	}
}

// Name returns the keystore's name
func (ks *BasicKeystore) Name() Name {
	return ks.name
}

// Type returns the keystore's key type
func (ks *BasicKeystore) Type() crypto.KeyType {
	return ks.typ
}

// Size returns the number of keys in the keystore
func (ks *BasicKeystore) Size() int {
	return len(ks.Keypairs())
}

// Insert adds a keypair to the keystore
func (ks *BasicKeystore) Insert(kp KeyPair) error {
	ks.lock.Lock()
	defer ks.lock.Unlock()

	if kp.Type() != ks.typ {
		return fmt.Errorf("%v, passed key type: %s, acceptable key type: %s", ErrKeyTypeNotSupported, kp.Type(), ks.typ)
	}

	pub := kp.Public()
	addr := crypto.PublicKeyToAddress(pub)
	ks.keys[addr] = kp
	return nil
}

// GetKeypair returns a keypair corresponding to the given public key, or nil if it doesn't exist
func (ks *BasicKeystore) GetKeypair(pub crypto.PublicKey) KeyPair {
	for _, key := range ks.keys {
		if bytes.Equal(key.Public().Encode(), pub.Encode()) {
			return key
		}
	}
	return nil
}

// GetKeypairFromAddress returns a keypair corresponding to the given address, or nil if it doesn't exist
func (ks *BasicKeystore) GetKeypairFromAddress(pub common.Address) KeyPair {
	ks.lock.RLock()
	defer ks.lock.RUnlock()
	return ks.keys[pub]
}

// PublicKeys returns all public keys in the keystore
func (ks *BasicKeystore) PublicKeys() (srkeys []crypto.PublicKey) {
	if ks.keys == nil {
		return srkeys
	}

	for _, key := range ks.keys {
		srkeys = append(srkeys, key.Public())
	}

	return srkeys
}

// Keypairs returns all keypairs in the keystore
func (ks *BasicKeystore) Keypairs() (srkeys []KeyPair) {
	if ks.keys == nil {
		return srkeys
	}

	for _, key := range ks.keys {
		srkeys = append(srkeys, key)
	}
	return srkeys
}
