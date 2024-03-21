// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	inmemory_trie "github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/require"
)

var (
	testGenesisHeader = &types.Header{
		Number:    0,
		StateRoot: inmemory_trie.EmptyHash,
		Digest:    types.NewDigest(),
	}
	testVote = &Vote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	testSignature   = [64]byte{1, 2, 3, 4}
	testAuthorityID = [32]byte{5, 6, 7, 8}
)

func newTestVoters(t *testing.T) []Voter {
	t.Helper()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	vs := []Voter{}
	for i, k := range kr.Keys {
		vs = append(vs, Voter{
			Key: *k.Public().(*ed25519.PublicKey),
			ID:  uint64(i),
		})
	}

	return vs
}
