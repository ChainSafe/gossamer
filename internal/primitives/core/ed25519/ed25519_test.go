package ed25519_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/stretchr/testify/require"
)

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

func TestSeedAndDeriveShouldWork(t *testing.T) {
	seedSlice, err := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	require.NoError(t, err)

	var seed [32]byte
	copy(seed[:], seedSlice)

	pair := ed25519.NewPairFromSeed(seed)
	require.Equal(t, pair.Seed(), seed)

	path := []crypto.DeriveJunction{crypto.NewDeriveJunction(crypto.DeriveJunctionHard{})}
	derived, _, err := pair.Derive(path, nil)
	require.NoError(t, err)

	expectedSlice, err := hex.DecodeString("ede3354e133f9c8e337ddd6ee5415ed4b4ffe5fc7d21e933f4930a3730e5b21c")
	require.NoError(t, err)
	var expected [32]byte
	copy(expected[:], expectedSlice)

	require.Equal(t, expected, derived.(ed25519.Pair).Seed())
}
