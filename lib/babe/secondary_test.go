// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestVerifySecondarySlotPlain(t *testing.T) {
	err := verifySecondarySlotPlain(0, 77, 1, Randomness{})
	require.NoError(t, err)

	err = verifySecondarySlotPlain(0, 77, 2, Randomness{})
	require.NoError(t, err)

	numAuths := 20
	numAuthorized := 0
	for i := 0; i < numAuths; i++ {
		err = verifySecondarySlotPlain(uint32(i), 77, numAuths, Randomness{})
		if err == nil {
			numAuthorized++
		}
	}

	require.Equal(t, 1, numAuthorized, "only one block producer should be authorized per secondary slot")
}

func createSecondaryVRFPreDigest(t *testing.T,
	keypair *sr25519.Keypair, index uint32,
	slot, epoch uint64, randomness Randomness,
) *types.BabeSecondaryVRFPreDigest {
	transcript := makeTranscript(randomness, slot, epoch)
	out, proof, err := keypair.VrfSign(transcript)
	require.NoError(t, err)

	return types.NewBabeSecondaryVRFPreDigest(index, slot, out, proof)
}

func TestVerifySecondarySlotVRF(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	slot := uint64(77)
	epoch := uint64(0)

	digest := createSecondaryVRFPreDigest(t, kp, 0, slot, epoch, Randomness{})

	ok, err := verifySecondarySlotVRF(digest, kp.Public().(*sr25519.PublicKey), epoch, 1, Randomness{})
	require.NoError(t, err)
	require.True(t, ok)

	numAuths := 20
	numAuthorized := 0
	for i := 0; i < numAuths; i++ {
		digest := createSecondaryVRFPreDigest(t, kp, uint32(i), slot, epoch, Randomness{})

		ok, err = verifySecondarySlotVRF(digest, kp.Public().(*sr25519.PublicKey), epoch, 1, Randomness{})
		if err == nil && ok {
			numAuthorized++
		}
	}

	require.Equal(t, 1, numAuthorized, "only one block producer should be authorized per secondary slot")
}
