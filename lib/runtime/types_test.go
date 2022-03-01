// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"io"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/stretchr/testify/require"
)

func TestBackgroundSignVerification(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)
	signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))

	signVerify.Start()

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	// Wait for background go routine to verify signature.
	time.Sleep(1 * time.Second)
	require.False(t, signVerify.IsInvalid())
}

func TestBackgroundSignVerificationMultipleStart(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)
	signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))

	for ii := 0; ii < 5; ii++ {
		require.False(t, signVerify.IsStarted())
		signVerify.Start()
		require.True(t, signVerify.IsStarted())

		for _, sig := range signs {
			signVerify.Add(sig)
		}
		require.True(t, signVerify.Finish())
		require.False(t, signVerify.IsStarted())
	}
}

func TestInvalidSignatureBatch(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)

	msg := []byte("ed25519")
	key, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	// Invalid Sign
	sigData, err := common.HexToBytes("0x90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301") //nolint:lll
	require.NoError(t, err)

	signature := &crypto.SignatureInfo{
		PubKey:     key.Public().Encode(),
		Sign:       sigData,
		Msg:        msg,
		VerifyFunc: ed25519.VerifySignature,
	}

	signs = append(signs, signature)

	signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))
	signVerify.Start()

	for _, sig := range signs {
		signVerify.Add(sig)
	}
	require.False(t, signVerify.Finish())
}

func TestValidSignatureBatch(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)
	signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))

	signVerify.Start()

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	require.True(t, signVerify.Finish())
}

func TestAllCryptoTypeSignature(t *testing.T) {
	edSignatures := generateEd25519Signatures(t, 1)

	srMsg := []byte("sr25519")
	srKey, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	srSig, err := srKey.Private().Sign(srMsg)
	require.NoError(t, err)

	srSignature := &crypto.SignatureInfo{
		PubKey:     srKey.Public().Encode(),
		Sign:       srSig,
		Msg:        srMsg,
		VerifyFunc: sr25519.VerifySignature,
	}

	blakeHash, err := common.Blake2bHash([]byte("secp256k1"))
	require.NoError(t, err)

	kp, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)

	secpSigData, err := kp.Sign(blakeHash.ToBytes())
	require.NoError(t, err)

	secpSigData = secpSigData[:len(secpSigData)-1] // remove recovery id
	secpSignature := &crypto.SignatureInfo{
		PubKey:     kp.Public().Encode(),
		Sign:       secpSigData,
		Msg:        blakeHash.ToBytes(),
		VerifyFunc: secp256k1.VerifySignature,
	}

	signVerify := crypto.NewSignatureVerifier(log.New(log.SetWriter(io.Discard)))

	signVerify.Start()

	signVerify.Add(edSignatures[0])
	signVerify.Add(srSignature)
	signVerify.Add(secpSignature)

	require.True(t, signVerify.Finish())
}
