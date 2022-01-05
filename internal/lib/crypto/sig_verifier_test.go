// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto_test

import (
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/internal/lib/crypto"
	"github.com/ChainSafe/gossamer/internal/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/internal/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/internal/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/internal/log"

	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	message := []byte("a225e8c75da7da319af6335e7642d473")

	edKeypair, err := ed25519.GenerateKeypair()
	require.NoError(t, err)
	edSign, err := edKeypair.Sign(message)
	require.NoError(t, err)

	secpKeypair, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)
	secpSign, err := secpKeypair.Sign(message)
	require.NoError(t, err)

	srKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	srSign, err := srKeypair.Sign(message)
	require.NoError(t, err)

	testCase := map[string]struct {
		expect             bool
		signaturesToVerify []*crypto.SignatureInfo
	}{
		"success": {
			expect: true,
			signaturesToVerify: []*crypto.SignatureInfo{
				0: {
					PubKey:     edKeypair.Public().Encode(),
					Sign:       edSign,
					Msg:        message,
					VerifyFunc: ed25519.VerifySignature,
				},
				1: {
					PubKey:     secpKeypair.Public().Encode(),
					Sign:       secpSign[:64],
					Msg:        message,
					VerifyFunc: secp256k1.VerifySignature,
				},
				2: {
					PubKey:     srKeypair.Public().Encode(),
					Sign:       srSign,
					Msg:        message,
					VerifyFunc: sr25519.VerifySignature,
				},
			},
		},
		"bad public key input": {
			expect: false,
			signaturesToVerify: []*crypto.SignatureInfo{
				0: {
					PubKey:     []byte{},
					Sign:       edSign,
					Msg:        message,
					VerifyFunc: ed25519.VerifySignature,
				},
				1: {
					PubKey:     []byte{},
					Sign:       srSign,
					Msg:        message,
					VerifyFunc: sr25519.VerifySignature,
				},
			},
		},
		"verification failed": {
			expect: false,
			signaturesToVerify: []*crypto.SignatureInfo{
				0: {
					PubKey:     edKeypair.Public().Encode(),
					Sign:       []byte{},
					Msg:        message,
					VerifyFunc: ed25519.VerifySignature,
				},
				1: {
					PubKey:     srKeypair.Public().Encode(),
					Sign:       []byte{},
					Msg:        message,
					VerifyFunc: sr25519.VerifySignature,
				},
				2: {
					PubKey:     secpKeypair.Public().Encode(),
					Sign:       []byte{},
					Msg:        message,
					VerifyFunc: secp256k1.VerifySignature,
				},
			},
		},
	}

	for name, value := range testCase {
		testCase := value
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))

			for _, sig := range testCase.signaturesToVerify {
				signVerify.Add(sig)
			}

			signVerify.Start()

			ok := signVerify.Finish()
			require.Equal(t, testCase.expect, ok)
		})
	}

}
