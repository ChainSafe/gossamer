// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import "github.com/ChainSafe/gossamer/lib/crypto"

// KeyPair is a key pair to sign messages and from which
// the public key and key type can be obtained.
type KeyPair interface {
	Sign(msg []byte) ([]byte, error)
	Publicer
	Typer
}

// PublicPrivater can return the private or public key
// from the keypair.
type PublicPrivater interface {
	Publicer
	Privater
}

// Publicer returns the public key of the keypair.
type Publicer interface {
	Public() crypto.PublicKey
}

// Privater returns the private key of the keypair.
type Privater interface {
	Private() crypto.PrivateKey
}

// Typer returns the key type.
type Typer interface {
	Type() crypto.KeyType
}
