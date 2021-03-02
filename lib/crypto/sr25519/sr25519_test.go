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
