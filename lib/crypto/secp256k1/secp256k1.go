// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package secp256k1

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ChainSafe/go-schnorrkel"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

// PublicKeyLength is the fixed Public Key Length
const PublicKeyLength int = 32

// PrivateKeyLength is the fixed Private Key Length
const PrivateKeyLength = 32

// SignatureLength is the fixed Signature Length
const SignatureLength = 64

// SignatureLengthRecovery is the length of a secp256k1 signature with recovery byte (used for ecrecover)
const SignatureLengthRecovery = 65

// MessageLength is the fixed Message Length
const MessageLength = 32

// Keypair holds the pub,pk keys
type Keypair struct {
	public  *PublicKey
	private *PrivateKey
}

// PublicKey struct for PublicKey
type PublicKey struct {
	key ecdsa.PublicKey
}

// VerifySignature verifies a signature given a public key and a message
func VerifySignature(publicKey, signature, message []byte) error {
	ok := secp256k1.VerifySignature(publicKey, message, signature)
	if ok {
		return nil
	}

	return fmt.Errorf("secp256k1: %w: for message 0x%x, signature 0x%x and public key 0x%x",
		crypto.ErrSignatureVerificationFailed, message, signature, publicKey)
}

// RecoverPublicKey returns the 64-byte uncompressed public key that created the given signature.
func RecoverPublicKey(msg, sig []byte) ([]byte, error) {
	// update recovery bit
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	return secp256k1.Ecrecover(msg, sig)
}

// RecoverPublicKeyCompressed returns the 33-byte compressed public key that signed the given message.
func RecoverPublicKeyCompressed(msg, sig []byte) ([]byte, error) {
	// update recovery bit
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pub, err := secp256k1.SigToPub(msg, sig)
	if err != nil {
		return nil, err
	}

	cpub := secp256k1.CompressPubkey(pub)
	return cpub, nil
}

// PrivateKey struct for PrivateKey
type PrivateKey struct {
	key ecdsa.PrivateKey
}

// NewKeypair will returned a Keypair from a PrivateKey
func NewKeypair(pk ecdsa.PrivateKey) *Keypair {
	pub := pk.Public()

	return &Keypair{
		public:  &PublicKey{key: *pub.(*ecdsa.PublicKey)},
		private: &PrivateKey{key: pk},
	}
}

// NewKeypairFromPrivate will return a Keypair for a PrivateKey
func NewKeypairFromPrivate(priv *PrivateKey) (*Keypair, error) {
	pub, err := priv.Public()
	if err != nil {
		return nil, err
	}

	return &Keypair{
		public:  pub.(*PublicKey),
		private: priv,
	}, nil
}

// NewKeypairFromMnenomic returns a new Keypair using the given mnemonic and password.
func NewKeypairFromMnenomic(mnemonic, password string) (*Keypair, error) {
	seed, err := schnorrkel.SeedFromMnemonic(mnemonic, password)
	if err != nil {
		return nil, err
	}
	return NewKeypairFromSeed(seed[:32])
}

// NewKeypairFromSeed generates a new secp256k1 keypair from a 32 byte seed
func NewKeypairFromSeed(seed []byte) (*Keypair, error) {
	// reference:
	// https://github.com/paritytech/polkadot-sdk/blob/7ca0d65f19497ac1c3c7ad6315f1a0acb2ca32f8/substrate/primitives/core/src/ecdsa.rs#L375-L391

	privateKey, err := NewPrivateKey(seed)
	if err != nil {
		return nil, fmt.Errorf("generating private key: %w", err)
	}

	return NewKeypair(privateKey.key), nil
}

// NewPublicKey returns an secp256k1 public key that consists of the input bytes
// Input length must be 32 bytes
func NewPublicKey(in []byte) (*PublicKey, error) {
	if len(in) != PublicKeyLength {
		return nil, fmt.Errorf("cannot create public key: input is not 32 bytes")
	}

	ecdsaPubKey, err := secp256k1.UnmarshalPubkey(in)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling public key: %w", err)
	}

	pubKey := PublicKey{key: *ecdsaPubKey}
	return &pubKey, nil
}

// NewPrivateKey will return a PrivateKey for a []byte
func NewPrivateKey(in []byte) (*PrivateKey, error) {
	if len(in) != PrivateKeyLength {
		return nil, errors.New("input to create secp256k1 private key is not 32 bytes")
	}
	priv := new(PrivateKey)
	err := priv.Decode(in)
	return priv, err
}

// NewKeypairFromPrivateKeyString returns a Keypair given a 0x prefixed private key string
func NewKeypairFromPrivateKeyString(in string) (*Keypair, error) {
	privBytes, err := common.HexToBytes(in)
	if err != nil {
		return nil, err
	}

	priv, err := NewPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	return NewKeypairFromPrivate(priv)
}

// GenerateKeypair will generate a Keypair
func GenerateKeypair() (*Keypair, error) {
	priv, err := secp256k1.GenerateKey()
	if err != nil {
		return nil, err
	}

	return NewKeypair(*priv), nil
}

// Type returns Secp256k1Type
func (*Keypair) Type() crypto.KeyType {
	return crypto.Secp256k1Type
}

// Sign will sign
func (kp *Keypair) Sign(msg []byte) ([]byte, error) {
	if len(msg) != MessageLength {
		return nil, errors.New("invalid message length: not 32 byte hash")
	}

	return secp256k1.Sign(msg, &kp.private.key)
}

// Public returns the pub key
func (kp *Keypair) Public() crypto.PublicKey {
	return kp.public
}

// Private returns pk
func (kp *Keypair) Private() crypto.PrivateKey {
	return kp.private
}

// Verify a msg
func (k *PublicKey) Verify(msg, sig []byte) (bool, error) {
	if len(sig) != SignatureLength {
		return false, errors.New("invalid signature length")
	}

	if len(msg) != MessageLength {
		return false, errors.New("invalid message length: not 32 byte hash")
	}

	return secp256k1.VerifySignature(k.Encode(), msg, sig), nil
}

// UnmarshalPubkey converts [65]byte to a secp256k1 public key.
func (k *PublicKey) UnmarshalPubkey(pub []byte) error {
	pubKey, err := secp256k1.UnmarshalPubkey(pub)
	if err != nil {
		return err
	}

	k.key = *pubKey
	return nil
}

// Encode will encode to []byte
func (k *PublicKey) Encode() []byte {
	return secp256k1.CompressPubkey(&k.key)
}

// Decode will decode to PublicKey key field
func (k *PublicKey) Decode(in []byte) error {
	pub, err := secp256k1.DecompressPubkey(in)
	if err != nil {
		return err
	}
	k.key = *pub
	return nil
}

// Address will return PublicKey Address
func (k *PublicKey) Address() common.Address {
	return crypto.PublicKeyToAddress(k)
}

// Hex will return PublicKey Hex
func (k *PublicKey) Hex() string {
	enc := k.Encode()
	h := hex.EncodeToString(enc)
	return "0x" + h
}

// Sign a message
func (pk *PrivateKey) Sign(msg []byte) ([]byte, error) {
	if len(msg) != MessageLength {
		return nil, errors.New("invalid message length: not 32 byte hash")
	}

	return secp256k1.Sign(msg, &pk.key)
}

// Public will return pub key
func (pk *PrivateKey) Public() (crypto.PublicKey, error) {
	return &PublicKey{
		key: *(pk.key.Public().(*ecdsa.PublicKey)),
	}, nil
}

// Encode will encode
func (pk *PrivateKey) Encode() []byte {
	return secp256k1.FromECDSA(&pk.key)
}

// Decode will decode
func (pk *PrivateKey) Decode(in []byte) error {
	key := secp256k1.ToECDSAUnsafe(in)
	pk.key = *key
	return nil
}

// Hex will return PrivateKey Hex
func (pk *PrivateKey) Hex() string {
	enc := pk.Encode()
	h := hex.EncodeToString(enc)
	return "0x" + h
}
