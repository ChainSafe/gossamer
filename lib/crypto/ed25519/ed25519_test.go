// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package ed25519

import (
	ed25519 "crypto/ed25519"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"

	bip39 "github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("helloworld")
	sig, _ := kp.Sign(msg)

	ok, err := Verify(kp.Public().(*PublicKey), msg, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Fail: did not verify ed25519 sig")
	}
}

func TestPublicKeys(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	kp2 := NewKeypair(ed25519.PrivateKey(*(kp.Private().(*PrivateKey))))
	if !reflect.DeepEqual(kp.Public(), kp2.Public()) {
		t.Fatal("Fail: pubkeys do not match")
	}
}

func TestEncodeAndDecodePrivateKey(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	enc := kp.Private().Encode()
	res := new(PrivateKey)
	err = res.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, kp.Private()) {
		t.Fatalf("Fail: got %x expected %x", res, kp.Private())
	}
}

func TestEncodeAndDecodePublicKey(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	enc := kp.Public().Encode()
	res := new(PublicKey)
	err = res.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, kp.Public()) {
		t.Fatalf("Fail: got %x expected %x", res, kp.Public())
	}
}

func TestNewKeypairFromMnenomic(t *testing.T) {
	entropy, err := bip39.NewEntropy(128)
	require.NoError(t, err)

	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)

	_, err = NewKeypairFromMnenomic(mnemonic, "")
	require.NoError(t, err)
}

func TestNewKeypairFromMnenomic_Again(t *testing.T) {
	mnemonic := "twist sausage october vivid neglect swear crumble hawk beauty fabric egg fragile"
	kp, err := NewKeypairFromMnenomic(mnemonic, "")
	require.NoError(t, err)

	expectedPubkey := common.MustHexToBytes("0xf56d9231e7b7badd3f1e10ad15ef8aa08b70839723d0a2d10d7329f0ea2b8c61")
	require.Equal(t, expectedPubkey, kp.Public().Encode())
}
