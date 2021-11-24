// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto_test

import (
	// "io"

	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestSigVerifierEd25519(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("Hello")
		key, err := ed25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.Signature{
			PubKey:     key.Public().Encode(),
			Sign:       sign,
			Msg:        msg,
			VerifyFunc: ed25519.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.True(t, ok)
}

func TestSigVerifierEd25519Fails(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("Hello")
		key, err := ed25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)
		if i == 1 {
			signs[i] = &crypto.Signature{
				PubKey:     []byte(""),
				Sign:       sign,
				Msg:        msg,
				VerifyFunc: ed25519.VerifySignature,
			}
			continue
		}
		signs[i] = &crypto.Signature{
			PubKey:     key.Public().Encode(),
			Sign:       []byte(""),
			Msg:        msg,
			VerifyFunc: ed25519.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.False(t, ok)
}

func TestSigVerifierSr25519(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("Hello")
		key, err := sr25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.Signature{
			PubKey:     key.Public().Encode(),
			Sign:       sign,
			Msg:        msg,
			VerifyFunc: sr25519.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.True(t, ok)
}

func TestSigVerifierSr25519Fails(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("Hello")
		key, err := sr25519.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.Signature{
			PubKey:     []byte(""),
			Sign:       sign,
			Msg:        msg,
			VerifyFunc: sr25519.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.False(t, ok)
}

func TestSigVerifierSecp256k1(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("a225e8c75da7da319af6335e7642d473")
		key, err := secp256k1.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.Signature{
			PubKey:     key.Public().Encode(),
			Sign:       sign[:64],
			Msg:        msg,
			VerifyFunc: secp256k1.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.True(t, ok)
}
func TestSigVerifierSecp256k1Fails(t *testing.T) {
	t.Helper()
	signs := make([]*crypto.Signature, 2)

	for i := 0; i < 2; i++ {
		msg := []byte("a225e8c75da7da319af6335e7642d473")
		key, err := secp256k1.GenerateKeypair()
		require.NoError(t, err)

		sign, err := key.Sign(msg)
		require.NoError(t, err)

		signs[i] = &crypto.Signature{
			PubKey:     []byte(""),
			Sign:       sign[:64],
			Msg:        msg,
			VerifyFunc: secp256k1.VerifySignature,
		}
	}
	signVerify := crypto.NewSignatureVerifier(log.New())

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	signVerify.Start()
	ok := signVerify.Finish()
	require.False(t, ok)
}
