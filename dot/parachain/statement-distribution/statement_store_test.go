package statementdistribution

import (
	"bytes"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestAlwaysProvidesFreshStatementsInOrder(t *testing.T) {
	validatorA := parachaintypes.ValidatorIndex(1)
	validatorB := parachaintypes.ValidatorIndex(2)
	candidateHash := parachaintypes.CandidateHash{Value: common.Hash(bytes.Repeat([]byte{0x42}, 32))}

	validStatement := parachaintypes.NewCompactValid(candidateHash)
	secondedStatement := parachaintypes.NewCompactSeconded(candidateHash)

	groups := newGroups([][]parachaintypes.ValidatorIndex{{validatorA, validatorB}}, 2)
	store := newStatementStore(groups)

	// Signed Valid by A
	var sigA [64]byte
	copy(sigA[:], bytes.Repeat([]byte{1}, 64))

	signedValidByA := &parachaintypes.SignedStatement{
		Payload:        *validStatement.ToEncodable(),
		ValidatorIndex: validatorA,
		Signature:      parachaintypes.ValidatorSignature(sigA),
	}
	_, err := store.insert(groups, signedValidByA, statementOriginRemote)
	require.NoError(t, err)

	// Signed Seconded by B
	var sigB [64]byte
	copy(sigB[:], bytes.Repeat([]byte{2}, 64))

	signedSecondedByB := &parachaintypes.SignedStatement{
		Payload:        *secondedStatement.ToEncodable(),
		ValidatorIndex: validatorB,
		Signature:      parachaintypes.ValidatorSignature(sigB),
	}
	_, err = store.insert(groups, signedSecondedByB, statementOriginRemote)
	require.NoError(t, err)

	// Regardless of the order statements are requested, we will get them in the order [B, A] because seconded statements must be first.
	vals := []parachaintypes.ValidatorIndex{validatorA, validatorB}
	statements := store.freshStatementsForBacking(vals, candidateHash)
	require.Len(t, statements, 2)
	cmp, err := statements[0].Payload.ToCompact()
	require.NoError(t, err)
	require.Equal(t, secondedStatement, cmp)

	cmp, err = statements[1].Payload.ToCompact()
	require.NoError(t, err)
	require.Equal(t, validStatement, cmp)

	vals = []parachaintypes.ValidatorIndex{validatorB, validatorA}
	statements = store.freshStatementsForBacking(vals, candidateHash)
	require.Len(t, statements, 2)
	cmp, err = statements[0].Payload.ToCompact()
	require.NoError(t, err)
	require.Equal(t, secondedStatement, cmp)

	cmp, err = statements[1].Payload.ToCompact()
	require.NoError(t, err)
	require.Equal(t, validStatement, cmp)
}
