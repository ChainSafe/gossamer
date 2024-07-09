// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/stretchr/testify/require"
)

func makePrecommit(t *testing.T,
	precommit grandpa.Precommit[hash.H256, uint64],
	round uint64,
	setID uint64,
	voter ed25519.Keyring,
) grandpa.SignedPrecommit[hash.H256, uint64, AuthoritySignature, AuthorityID] {
	t.Helper()
	msg := grandpa.NewMessage(precommit)
	encoded := LocalizedPayload(Precommit, RoundNumber(round), SetID(setID), msg)
	signature := voter.Sign(encoded)

	return grandpa.SignedPrecommit[hash.H256, uint64, AuthoritySignature, AuthorityID]{
		Precommit: precommit,
		Signature: signature,
		ID:        voter.Pair().Public().(ced25519.Public),
	}
}

func TestCheckMessageSignature(t *testing.T) {
	precommit := grandpa.Precommit[hash.H256, uint64]{
		TargetHash:   hash.H256("a"),
		TargetNumber: 1,
	}
	signedPrecommit := makePrecommit(t, precommit, 1, 1, ed25519.Alice)
	valid := CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), signedPrecommit.ID, signedPrecommit.Signature, 1, 1)
	require.True(t, valid)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), signedPrecommit.ID, signedPrecommit.Signature, 2, 1)
	require.False(t, valid)

	signedPrecommit = makePrecommit(t, precommit, 2, 1, ed25519.Alice)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), signedPrecommit.ID, signedPrecommit.Signature, 2, 1)
	require.True(t, valid)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), signedPrecommit.ID, signedPrecommit.Signature, 1, 1)
	require.False(t, valid)

	signedPrecommit = makePrecommit(t, precommit, 3, 3, ed25519.Bob)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), signedPrecommit.ID, signedPrecommit.Signature, 3, 3)
	require.True(t, valid)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), ed25519.Bob.Pair().Public().(ced25519.Public), signedPrecommit.Signature, 3, 3)
	require.True(t, valid)
	valid = CheckMessageSignature[hash.H256, uint64](
		grandpa.NewMessage(precommit), ed25519.Alice.Pair().Public().(ced25519.Public), signedPrecommit.Signature, 3, 3)
	require.False(t, valid)
}
