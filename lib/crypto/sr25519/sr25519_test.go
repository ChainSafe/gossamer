// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sr25519

import (
	"crypto/rand"
	"testing"

	bip39 "github.com/cosmos/go-bip39"
	"github.com/gtank/merlin"
	"github.com/stretchr/testify/require"
)

func TestNewKeypairFromSeed(t *testing.T) {
	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	require.NoError(t, err)

	kp, err := NewKeypairFromSeed(seed)
	require.NoError(t, err)
	require.NotNil(t, kp.public)
	require.NotNil(t, kp.private)

	seed = make([]byte, 20)
	_, err = rand.Read(seed)
	require.NoError(t, err)
	kp, err = NewKeypairFromSeed(seed)
	require.Nil(t, kp)
	require.Error(t, err, "cannot generate key from seed: seed is not 32 bytes long")
}

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	msg := []byte("helloworld")
	sig, err := kp.Sign(msg)
	require.NoError(t, err)

	pub := kp.Public().(*PublicKey)
	ok, err := pub.Verify(msg, sig)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestPublicKeys(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	priv := kp.Private().(*PrivateKey)
	kp2, err := NewKeypair(priv.key)
	require.NoError(t, err)
	require.Equal(t, kp.Public(), kp2.Public())
}

func TestEncodeAndDecodePrivateKey(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	enc := kp.Private().Encode()
	res := new(PrivateKey)
	err = res.Decode(enc)
	require.NoError(t, err)

	exp := kp.Private().(*PrivateKey).key.Encode()
	require.Equal(t, exp, res.key.Encode())
}

func TestEncodeAndDecodePublicKey(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	enc := kp.Public().Encode()
	res := new(PublicKey)
	err = res.Decode(enc)
	require.NoError(t, err)

	exp := kp.Public().(*PublicKey).key.Encode()
	require.Equal(t, exp, res.key.Encode())
}

func TestVrfSignAndVerify(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	transcript := merlin.NewTranscript("helloworld")
	out, proof, err := kp.VrfSign(transcript)
	require.NoError(t, err)

	pub := kp.Public().(*PublicKey)
	transcript2 := merlin.NewTranscript("helloworld")
	ok, err := pub.VrfVerify(transcript2, out, proof)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestSignAndVerify_Deprecated(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	msg := []byte("helloworld")
	sig, err := kp.Sign(msg)
	require.NoError(t, err)

	pub := kp.Public().(*PublicKey)
	ok, err := pub.VerifyDeprecated(msg, sig)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestNewKeypairFromMnenomic(t *testing.T) {
	entropy, err := bip39.NewEntropy(128)
	require.NoError(t, err)

	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)

	_, err = NewKeypairFromMnenomic(mnemonic, "")
	require.NoError(t, err)
}
