// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"reflect"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// private keys generated using `subkey inspect //Name`
var sr25519PrivateKeys = []string{
	"0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a",
	"0x398f0c28f98885e046333d4a41c19cee4c37368a9832c6502f6cfd182e2aef89",
	"0xbc1ede780f784bb6991a585e4f6e61522c14e1cae6ad0895fb57b9a205a8f938",
	"0x868020ae0687dda7d57565093a69090211449845a7e11453612800b663307246",
	"0x786ad0e2df456fe43dd1f91ebca22e235bc162e0bb8d53c633e8c85b2af68b7a",
	"0x42438b7883391c05512a938e36c2df0131e088b3756d6aa7a755fbff19d2f842",
	"0xcdb035129162df39b70e604ab75162084e176f48897cdafb7d72c4a542a86dda",
	"0x51079fc9e1817f8d4f245d66b325a94d9cafdb8691acbfe85415dce3ae7a62b9",
	"0x7c04eea9d31ce0d9ee256d7c561dc29f20d1119a125e95713c967dcd8d14f22d",
}

// Keyring represents a test keyring
type Keyring interface {
	Alice() crypto.Keypair
	Bob() crypto.Keypair
	Charlie() crypto.Keypair
	Dave() crypto.Keypair
	Eve() crypto.Keypair
	Ferdie() crypto.Keypair
	George() crypto.Keypair
	Heather() crypto.Keypair
	Ian() crypto.Keypair
}

// Sr25519Keyring represents a test keyring
type Sr25519Keyring struct {
	KeyAlice   *sr25519.Keypair
	KeyBob     *sr25519.Keypair
	KeyCharlie *sr25519.Keypair
	KeyDave    *sr25519.Keypair
	KeyEve     *sr25519.Keypair
	KeyFerdie  *sr25519.Keypair
	KeyGeorge  *sr25519.Keypair
	KeyHeather *sr25519.Keypair
	KeyIan     *sr25519.Keypair

	Keys []*sr25519.Keypair
}

// NewSr25519Keyring returns an initialised sr25519 Keyring
func NewSr25519Keyring() (*Sr25519Keyring, error) {
	kr := new(Sr25519Keyring)
	v := reflect.ValueOf(kr).Elem()
	kr.Keys = make([]*sr25519.Keypair, v.NumField()-1)

	for i := 0; i < v.NumField()-1; i++ {
		who := v.Field(i)
		h, err := common.HexToBytes(sr25519PrivateKeys[i])
		if err != nil {
			return nil, err
		}

		kp, err := sr25519.NewKeypairFromSeed(h)
		if err != nil {
			return nil, err
		}

		who.Set(reflect.ValueOf(kp))

		kr.Keys[i] = kp
	}

	return kr, nil
}

// Alice returns Alice's key
func (kr *Sr25519Keyring) Alice() crypto.Keypair {
	return kr.KeyAlice
}

// Bob returns Bob's key
func (kr *Sr25519Keyring) Bob() crypto.Keypair {
	return kr.KeyBob
}

// Charlie returns Charlie's key
func (kr *Sr25519Keyring) Charlie() crypto.Keypair {
	return kr.KeyCharlie
}

// Dave returns Dave's key
func (kr *Sr25519Keyring) Dave() crypto.Keypair {
	return kr.KeyDave
}

// Eve returns Eve's key
func (kr *Sr25519Keyring) Eve() crypto.Keypair {
	return kr.KeyEve
}

// Ferdie returns Ferdie's key
func (kr *Sr25519Keyring) Ferdie() crypto.Keypair {
	return kr.KeyFerdie
}

// George returns George's key
func (kr *Sr25519Keyring) George() crypto.Keypair {
	return kr.KeyGeorge
}

// Heather returns Heather's key
func (kr *Sr25519Keyring) Heather() crypto.Keypair {
	return kr.KeyHeather
}

// Ian returns Ian's key
func (kr *Sr25519Keyring) Ian() crypto.Keypair {
	return kr.KeyIan
}

var ed25519PrivateKeys = []string{
	"0xabf8e5bdbe30c65656c0a3cbd181ff8a56294a69dfedd27982aace4a76909115",
	"0x3b7b60af2abcd57ba401ab398f84f4ca54bd6b2140d2503fbcf3286535fe3ff1",
	"0x072c02fa1409dc37e03a4ed01703d4a9e6bba9c228a49a00366e9630a97cba7c",
	"0x771f47d3caf8a2ee40b0719e1c1ecbc01d73ada220cf08df12a00453ab703738",
	"0xbef5a3cd63dd36ab9792364536140e5a0cce6925969940c431934de056398556",
	"0x1441e38eb309b66e9286867a5cd05902b05413eb9723a685d4d77753d73d0a1d",
	"0x583b887078cbae4b6ac6fbee324c3d2c16f3a1f8bf18f0d234de3ac33baa4470",
	"0xb8f3de627932e28914f3bc4bc3d7d2fc95c1f95c7915343d79df68d8250de180",
	"0xfd9f15cac5ffd14ed08914c200b1744ab00bdddf45e86cd13ccf9585ffa0e3ce",
}

// Ed25519Keyring represents a test ed25519 keyring
type Ed25519Keyring struct {
	KeyAlice   *ed25519.Keypair
	KeyBob     *ed25519.Keypair
	KeyCharlie *ed25519.Keypair
	KeyDave    *ed25519.Keypair
	KeyEve     *ed25519.Keypair
	KeyFerdie  *ed25519.Keypair
	KeyGeorge  *ed25519.Keypair
	KeyHeather *ed25519.Keypair
	KeyIan     *ed25519.Keypair

	Keys []*ed25519.Keypair
}

// NewEd25519Keyring returns an initialised ed25519 Keyring
func NewEd25519Keyring() (*Ed25519Keyring, error) {
	kr := new(Ed25519Keyring)
	v := reflect.ValueOf(kr).Elem()
	kr.Keys = make([]*ed25519.Keypair, v.NumField()-1)

	for i := 0; i < v.NumField()-1; i++ {
		who := v.Field(i)
		kp, err := ed25519.NewKeypairFromPrivateKeyString(ed25519PrivateKeys[i])
		if err != nil {
			return nil, err
		}
		who.Set(reflect.ValueOf(kp))

		kr.Keys[i] = kp
	}

	return kr, nil
}

// Alice returns Alice's key
func (kr *Ed25519Keyring) Alice() crypto.Keypair {
	return kr.KeyAlice
}

// Bob returns Bob's key
func (kr *Ed25519Keyring) Bob() crypto.Keypair {
	return kr.KeyBob
}

// Charlie returns Charlie's key
func (kr *Ed25519Keyring) Charlie() crypto.Keypair {
	return kr.KeyCharlie
}

// Dave returns Dave's key
func (kr *Ed25519Keyring) Dave() crypto.Keypair {
	return kr.KeyDave
}

// Eve returns Eve's key
func (kr *Ed25519Keyring) Eve() crypto.Keypair {
	return kr.KeyEve
}

// Ferdie returns Ferdie's key
func (kr *Ed25519Keyring) Ferdie() crypto.Keypair {
	return kr.KeyFerdie
}

// George returns George's key
func (kr *Ed25519Keyring) George() crypto.Keypair {
	return kr.KeyGeorge
}

// Heather returns Heather's key
func (kr *Ed25519Keyring) Heather() crypto.Keypair {
	return kr.KeyHeather
}

// Ian returns Ian's key
func (kr *Ed25519Keyring) Ian() crypto.Keypair {
	return kr.KeyIan
}
