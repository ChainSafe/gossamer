// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto

import (
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/btcsuite/btcutil/base58"
	bip39 "github.com/cosmos/go-bip39"
	"golang.org/x/crypto/blake2b"
)

// KeyType str
type KeyType = string

// Ed25519Type ed25519
const Ed25519Type KeyType = "ed25519"

// Sr25519Type sr25519
const Sr25519Type KeyType = "sr25519"

// Secp256k1Type secp256k1
const Secp256k1Type KeyType = "secp256k1"

// UnknownType is used by the GenericKeystore
const UnknownType KeyType = "unknown"

// PublicKey interface
type PublicKey interface {
	Verify(msg, sig []byte) (bool, error)
	Encode() []byte
	Decode([]byte) error
	Address() common.Address
	Hex() string
}

// PrivateKey interface
type PrivateKey interface {
	Sign(msg []byte) ([]byte, error)
	Public() (PublicKey, error)
	Encode() []byte
	Decode([]byte) error
	Hex() string
}

var ss58Prefix = []byte("SS58PRE")

// PublicKeyToAddress returns an ss58 address given a PublicKey
// see: https://github.com/paritytech/substrate/wiki/External-Address-Format-(SS58)
// also see: https://github.com/paritytech/substrate/blob/master/primitives/core/src/crypto.rs#L275
func PublicKeyToAddress(pub PublicKey) common.Address {
	enc := append([]byte{42}, pub.Encode()...)
	return publicKeyBytesToAddress(enc)
}

func publicKeyBytesToAddress(b []byte) common.Address {
	hasher, err := blake2b.New(64, nil)
	if err != nil {
		return ""
	}
	_, err = hasher.Write(append(ss58Prefix, b...))
	if err != nil {
		return ""
	}
	checksum := hasher.Sum(nil)
	return common.Address(base58.Encode(append(b, checksum[:2]...)))
}

// PublicAddressToByteArray returns []byte address for given PublicKey Address
func PublicAddressToByteArray(add common.Address) []byte {
	if add == "" {
		return nil
	}
	k := base58.Decode(string(add))
	return k[1:33]
}

// NewBIP39Mnemonic returns a new BIP39-compatible mnemonic
func NewBIP39Mnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return "", err
	}

	return bip39.NewMnemonic(entropy)
}
