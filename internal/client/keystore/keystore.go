// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

type PublicKey struct {
	Key []byte
	crypto.KeyTypeID
}

// KeyStore generates, stores and provides access to secret keys.
type KeyStore interface {
	// Generate an ed25519 signature for a given message.
	//
	// Receives [crypto.KeyTypeID] and an [ed25519.Public] key to be able to map them to a private key that exists in
	// the keystore.
	//
	// Returns an [ed25519.Signature] or nil in case the given keyType and public combination doesn't exist in the
	// keystore. An error will be returned if generating the signature itself failed.
	Ed25519Sign(keyType crypto.KeyTypeID, public ed25519.Public, msg []byte) (*ed25519.Signature, error)

	// Checks if the private keys for the given public key and key type combinations exist.
	//
	// Returns true iff all private keys could be found.
	HasKeys(publicKeys []PublicKey) bool
}
