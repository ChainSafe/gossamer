// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package ed25519_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/stretchr/testify/require"
)

func mustHexDecodeString32(t *testing.T, s string) [32]byte {
	t.Helper()
	seedSlice, err := hex.DecodeString(s)
	require.NoError(t, err)

	var seed [32]byte
	copy(seed[:], seedSlice)
	return seed
}
func mustHexDecodeString64(t *testing.T, s string) [64]byte {
	t.Helper()
	seedSlice, err := hex.DecodeString(s)
	require.NoError(t, err)

	var seed [64]byte
	copy(seed[:], seedSlice)
	return seed
}

func TestDefaultPhraseShouldBeUsed(t *testing.T) {
	pair, err := ed25519.NewPairFromString("//Alice///password", nil)
	require.NoError(t, err)

	pass := "password"
	pair1, err := ed25519.NewPairFromString(
		fmt.Sprintf("%s//Alice", crypto.DevPhrase), &pass,
	)
	require.NoError(t, err)

	require.Equal(t, pair, pair1)
}

func TestNewPairFromString_DifferentAliases(t *testing.T) {
	pair, err := ed25519.NewPairFromString("//Alice///password", nil)
	require.NoError(t, err)

	pair1, err := ed25519.NewPairFromString("//Bob///password", nil)
	require.NoError(t, err)

	require.NotEqual(t, pair, pair1)
}

func TestSeedAndDeriveShouldWork(t *testing.T) {
	seed := mustHexDecodeString32(t, "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	pair := ed25519.NewPairFromSeed(seed)
	require.Equal(t, pair.Seed(), seed)

	path := []crypto.DeriveJunction{crypto.NewDeriveJunction(crypto.DeriveJunctionHard{})}
	derived, _, err := pair.Derive(path, nil)
	require.NoError(t, err)

	expected := mustHexDecodeString32(t, "ede3354e133f9c8e337ddd6ee5415ed4b4ffe5fc7d21e933f4930a3730e5b21c")
	require.Equal(t, expected, derived.(ed25519.Pair).Seed())
}

func TestVectorShouldWork(t *testing.T) {
	seed := mustHexDecodeString32(t, "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	expected := mustHexDecodeString32(t, "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")

	pair := ed25519.NewPairFromSeed(seed)
	public := pair.Public()
	require.Equal(t, public, ed25519.NewPublicFromRaw(expected))

	signature := mustHexDecodeString64(t, "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
	message := []byte("")
	require.Equal(t, ed25519.NewSignatureFromRaw(signature), pair.Sign(message))
	require.True(t, public.Verify(signature, message))
}

func TestVectorByStringShouldWork(t *testing.T) {
	pair, err := ed25519.NewPairFromString("0x9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60", nil)
	require.NoError(t, err)
	public := pair.Public()
	require.Equal(t, ed25519.NewPublicFromRaw(
		mustHexDecodeString32(t, "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"),
	), public)

	signature := mustHexDecodeString64(t, "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
	message := []byte("")
	require.Equal(t, ed25519.NewSignatureFromRaw(signature), pair.Sign(message))
	require.True(t, public.Verify(signature, message))
}

func TestGeneratedPairShouldWork(t *testing.T) {
	pair, _ := ed25519.NewGeneratedPair()
	public := pair.Public()
	message := []byte("Something important")
	signature := pair.Sign(message)
	require.True(t, public.Verify(signature, message))
	require.False(t, public.Verify(signature, []byte("Something else")))
}

func TestSeededPairShouldWork(t *testing.T) {
	pair := ed25519.NewPairFromSeedSlice([]byte("12345678901234567890123456789012"))
	public := pair.Public()
	require.Equal(t, public, ed25519.NewPublicFromRaw(
		mustHexDecodeString32(t, "2f8c6129d816cf51c374bc7f08c3e63ed156cf78aefb4a6550d97b87997977ee"),
	))
	message := mustHexDecodeString32(t, "2f8c6129d816cf51c374bc7f08c3e63ed156cf78aefb4a6550d97b87997977ee")
	signature := pair.Sign(message[:])
	require.True(t, public.Verify(signature, message[:]))
	require.False(t, public.Verify(signature, []byte("Other Message")))
}

func TestGenerateWithPhraseRecoveryPossible(t *testing.T) {
	pair1, phrase, _ := ed25519.NewGeneratedPairWithPhrase(nil)
	pair2, _, err := ed25519.NewPairFromPhrase(phrase, nil)
	require.NoError(t, err)
	require.Equal(t, pair1.Public(), pair2.Public())
}

func TestGenerateWithPasswordPhraseRecoverPossible(t *testing.T) {
	password := "password"
	pair1, phrase, _ := ed25519.NewGeneratedPairWithPhrase(&password)
	pair2, _, err := ed25519.NewPairFromPhrase(phrase, &password)
	require.NoError(t, err)
	require.Equal(t, pair1.Public(), pair2.Public())
}

func TestPasswordDoesSomething(t *testing.T) {
	password := "password"
	pair1, phrase, _ := ed25519.NewGeneratedPairWithPhrase(&password)
	pair2, _, err := ed25519.NewPairFromPhrase(phrase, nil)
	require.NoError(t, err)
	require.NotEqual(t, pair1.Public(), pair2.Public())
}
