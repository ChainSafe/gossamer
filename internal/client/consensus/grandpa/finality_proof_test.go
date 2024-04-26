// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa/mocks"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Check GRANDPA proof-of-finality for the given block.
//
// Returns the vector of headers that MUST be validated + imported
// AND if at least one of those headers is invalid, all other MUST be considered invalid.
func checkFinalityProof[Hash runtime.Hash, N runtime.Number](
	currentSetID uint64,
	currentAuthorities pgrandpa.AuthorityList,
	remoteProof []byte,
) (FinalityProof[Hash, N], error) {
	proof := FinalityProof[Hash, N]{}
	err := scale.Unmarshal(remoteProof, &proof)
	if err != nil {
		return FinalityProof[Hash, N]{}, fmt.Errorf("failed to decode finality proof %s", err)
	}

	justification := GrandpaJustification[Hash, N]{}
	err = scale.Unmarshal(proof.Justification, &justification)
	if err != nil {
		return FinalityProof[Hash, N]{}, fmt.Errorf("error decoding justification for header %s", err)
	}

	err = justification.Verify(currentSetID, currentAuthorities)
	if err != nil {
		return FinalityProof[Hash, N]{}, err
	}

	return proof, nil
}

func TestFinalityProof_FailsIfNoMoreLastFinalizedBlocks(t *testing.T) {
	dummyInfo := blockchain.Info[hash.H256, uint32]{
		FinalizedNumber: 4,
	}
	mockBlockchain := mocks.NewBlockchainBackend[hash.H256, uint32](t)
	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()

	mockBackend := mocks.NewBackend[hash.H256, uint32, runtime.BlakeTwo256](t)
	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Once()

	// The last finalized block is 4, so we cannot provide further justifications.
	authoritySetChanges := AuthoritySetChanges[uint32]{}
	_, err := proveFinality[hash.H256, uint32, runtime.BlakeTwo256](
		mockBackend,
		authoritySetChanges,
		5,
		true)
	require.ErrorIs(t, err, errBlockNotYetFinalized)
}

func TestFinalityProof_IsNoneIfNoJustificationKnown(t *testing.T) {
	dummyInfo := blockchain.Info[hash.H256, uint32]{
		FinalizedNumber: 4,
	}
	dummyHash := hash.H256("dummyHash")
	mockBlockchain := mocks.NewBlockchainBackend[hash.H256, uint32](t)
	mockBlockchain.EXPECT().Info().Return(dummyInfo).Once()
	mockBlockchain.EXPECT().ExpectBlockHashFromID(uint32(4)).Return(dummyHash, nil).Once()
	mockBlockchain.EXPECT().Justifications(dummyHash).Return(nil, nil).Once()

	mockBackend := mocks.NewBackend[hash.H256, uint32, runtime.BlakeTwo256](t)
	mockBackend.EXPECT().Blockchain().Return(mockBlockchain).Times(3)

	authoritySetChanges := AuthoritySetChanges[uint32]{}
	authoritySetChanges.append(0, 4)

	// Block 4 is finalized without justification
	// => we can't prove finality of 3
	proofOf3, err := proveFinality[hash.H256, uint32, runtime.BlakeTwo256](
		mockBackend,
		authoritySetChanges,
		3,
		true,
	)
	require.NoError(t, err)
	require.Nil(t, proofOf3)
}

func TestFinalityProof_CheckFailsWhenProofDecodeFails(t *testing.T) {
	// When we can't decode proof from Vec<u8>
	_, err := checkFinalityProof[hash.H256, uint32](
		1,
		pgrandpa.AuthorityList{},
		[]byte{42},
	)
	require.NotNil(t, err)
	require.ErrorContains(t, err, "failed to decode finality proof")
}

func TestFinalityProof_CheckFailsWhenProofIsEmpty(t *testing.T) {
	// When decoded proof has zero length
	authorityList := pgrandpa.AuthorityList{}
	grandpaJustification := GrandpaJustification[hash.H256, uint32]{}
	encJustification, err := scale.Marshal(grandpaJustification)
	require.NoError(t, err)
	_, err = checkFinalityProof[hash.H256, uint32](
		1,
		authorityList,
		encJustification,
	)
	require.NotNil(t, err)
}

func TestFinalityProof_CheckFailsWithIncompleteJustification(t *testing.T) {
	authorityList := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     newTestPublic(t, 3),
			AuthorityWeight: 1,
		},
	}

	// Create a commit without precommits
	commit := pgrandpa.Commit[hash.H256, uint32]{
		TargetHash:   "hash7",
		TargetNumber: 7,
	}

	grandpaJust := GrandpaJustification[hash.H256, uint32]{
		Justification: pgrandpa.GrandpaJustification[hash.H256, uint32]{
			Round:  8,
			Commit: commit,
		},
	}

	finalityProof := FinalityProof[hash.H256, uint32]{
		Block:         "hash2",
		Justification: scale.MustMarshal(grandpaJust),
	}

	_, err := checkFinalityProof[hash.H256, uint32](
		1,
		authorityList,
		scale.MustMarshal(finalityProof),
	)
	require.ErrorIs(t, err, errBadJustification)
}

func createCommit[H runtime.Hash, N runtime.Number](
	t *testing.T,
	block runtime.Block[N, H],
	round uint64,
	setID pgrandpa.SetID,
	auth []ed25519.Keyring,
) pgrandpa.Commit[H, N] {
	t.Helper()

	var precommits []grandpa.SignedPrecommit[H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]

	for _, voter := range auth {
		precommit := grandpa.Precommit[H, N]{
			TargetHash:   block.Hash(),
			TargetNumber: block.Header().Number(),
		}
		msg := grandpa.NewMessage(precommit)
		encoded := pgrandpa.LocalizedPayload(pgrandpa.RoundNumber(round), setID, msg)
		signature := voter.Sign(encoded)

		signedPrecommit := grandpa.SignedPrecommit[H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]{
			Precommit: precommit,
			Signature: signature,
			ID:        voter.Pair().Public().(ced25519.Public),
		}
		precommits = append(precommits, signedPrecommit)
	}

	return pgrandpa.Commit[H, N]{
		TargetHash:   block.Hash(),
		TargetNumber: block.Header().Number(),
		Precommits:   precommits,
	}
}

func newHeader(number uint64) *generic.Header[uint64, hash.H256, runtime.BlakeTwo256] {
	// var defaultHash = [32]byte{}
	var parentHash = hash.H256("")
	switch number {
	case 0:
	default:
		parentHash = newHeader(number - 1).Hash()
	}
	header := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		number, hash.H256(""), hash.H256(""), parentHash, runtime.Digest{})
	return &header
}

func TestNewHeader(t *testing.T) {
	header := newHeader(2)
	hash := header.Hash()
	require.Equal(
		t, hash.Bytes(),
		[]byte{
			26, 124, 34, 215, 232, 187, 104, 22, 29, 232, 40, 118, 219, 37, 121, 10, 210,
			220, 188, 99, 242, 208, 233, 23, 243, 102, 164, 192, 220, 154, 183, 105,
		},
	)
}

func TestFinalityProof_CheckWorksWithCorrectJustification(t *testing.T) {
	alice := ed25519.Alice
	var setID pgrandpa.SetID = 1
	var round uint64 = 8
	var block = generic.NewBlock[uint64, hash.H256, runtime.BlakeTwo256](newHeader(7), nil)
	commit := createCommit(t, block, round, setID, []ed25519.Keyring{alice})

	var client blockchain.HeaderBackend[hash.H256, uint64]

	grandpaJust, err := NewJustificationFromCommit[hash.H256, uint64](client, round, commit)
	require.NoError(t, err)

	finalityProof := FinalityProof[hash.H256, uint64]{
		Block:          newHeader(2).Hash(),
		Justification:  scale.MustMarshal(grandpaJust),
		UnknownHeaders: nil,
	}

	authorityList := pgrandpa.AuthorityList{
		pgrandpa.AuthorityIDWeight{
			AuthorityID:     alice.Pair().Public().(pgrandpa.AuthorityID),
			AuthorityWeight: 1,
		},
	}

	proof, err := checkFinalityProof[hash.H256, uint64](
		uint64(setID), authorityList, scale.MustMarshal(finalityProof),
	)
	require.NoError(t, err)
	require.Equal(t, finalityProof, proof)
}

func TestFinalityProof_UsingAuthoritySetChangesFailsWithUndefinedStart(t *testing.T) {
	// let (_, backend, _) = test_blockchain(8, &[4, 5, 8]);
	info := blockchain.Info[hash.H256, uint64]{
		FinalizedNumber: 8,
	}
	mockBlockchainBackend := mocks.NewBlockchainBackend[hash.H256, uint64](t)
	mockBlockchainBackend.EXPECT().Info().Return(info).Once()

	mockBackend := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256](t)
	mockBackend.EXPECT().Blockchain().Return(mockBlockchainBackend).Once()

	// We are missing the block for the preceding set the start is not well-defined.
	authoritySetChanges := AuthoritySetChanges[uint64]{}
	authoritySetChanges.append(1, 8)

	_, err := proveFinality[hash.H256, uint64, runtime.BlakeTwo256](
		mockBackend,
		authoritySetChanges,
		6,
		true,
	)
	require.ErrorIs(t, err, errBlockNotInAuthoritySetChanges)
}

func TestFinalityProof_UsingAuthoritySetChangesWorks(t *testing.T) {
	var client blockchain.HeaderBackend[hash.H256, uint64]

	// let (client, backend, blocks) = test_blockchain(8, &[4, 5]);
	block7 := generic.NewBlock[uint64, hash.H256, runtime.BlakeTwo256](newHeader(7), nil)
	block8 := generic.NewBlock[uint64, hash.H256, runtime.BlakeTwo256](newHeader(8), nil)

	round := uint64(8)
	commit := createCommit(t, block8, round, 1, []ed25519.Keyring{ed25519.Alice})
	grandpaJust8, err := NewJustificationFromCommit(client, round, commit)
	require.NoError(t, err)

	// client
	// .finalize_block(block8.hash(), Some((ID, grandpa_just8.encode().clone())))
	// .unwrap();
	blockchainBackend := mocks.NewBlockchainBackend[hash.H256, uint64](t)
	blockchainBackend.EXPECT().Info().Return(blockchain.Info[hash.H256, uint64]{
		FinalizedNumber: 8,
	})
	blockchainBackend.EXPECT().ExpectBlockHashFromID(uint64(8)).Return(block8.Hash(), nil)
	blockchainBackend.EXPECT().ExpectHeader(block8.Hash()).Return(block8.Header(), nil)

	justification := runtime.Justification{
		ConsensusEngineID:    pgrandpa.GrandpaEngineID,
		EncodedJustification: scale.MustMarshal(grandpaJust8),
	}
	blockchainBackend.EXPECT().Justifications(block8.Hash()).Return(&runtime.Justifications{justification}, nil)

	blockchainBackend.EXPECT().ExpectBlockHashFromID(uint64(7)).Return(block7.Hash(), nil)
	blockchainBackend.EXPECT().ExpectHeader(block7.Hash()).Return(block7.Header(), nil)

	backend := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256](t)
	backend.EXPECT().Blockchain().Return(blockchainBackend)

	// Authority set change at block 8, so the justification stored there will be used in the
	// FinalityProof for block 6
	authoritySetChanges := AuthoritySetChanges[uint64]{}
	authoritySetChanges.append(0, 5)
	authoritySetChanges.append(1, 8)

	proofOf6, err := proveFinality[hash.H256, uint64, runtime.BlakeTwo256](backend, authoritySetChanges, 6, true)
	require.NoError(t, err)
	require.NotNil(t, proofOf6)

	assert.Equal(t, FinalityProof[hash.H256, uint64]{
		Block:          block8.Hash(),
		Justification:  scale.MustMarshal(grandpaJust8),
		UnknownHeaders: []runtime.Header[uint64, hash.H256]{block7.Header(), block8.Header()},
	}, *proofOf6)

	proofOf6WithoutUnknown, err := proveFinality[hash.H256, uint64, runtime.BlakeTwo256](
		backend, authoritySetChanges, 6, false)
	require.NoError(t, err)
	require.NotNil(t, proofOf6WithoutUnknown)

	assert.Equal(t, FinalityProof[hash.H256, uint64]{
		Block:          block8.Hash(),
		Justification:  scale.MustMarshal(grandpaJust8),
		UnknownHeaders: nil,
	}, *proofOf6WithoutUnknown)
}

func TestFinalityProof_InLastSetFailsWithoutLatest(t *testing.T) {
	blockchainBackend := mocks.NewBlockchainBackend[hash.H256, uint64](t)
	blockchainBackend.EXPECT().Info().Return(blockchain.Info[hash.H256, uint64]{
		FinalizedNumber: 8,
	}).Once()

	backend := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256](t)
	backend.EXPECT().Blockchain().Return(blockchainBackend).Once()
	backend.EXPECT().GetAux(bestJustification).Return(nil, nil).Once()

	// No recent authority set change, so we are in the authoritySetChangeIDLatest set, and we will try to pickup
	// the best stored justification, for which there is none in this case.
	authoritySetChanges := AuthoritySetChanges[uint64]{}
	authoritySetChanges.append(0, 5)

	proof, err := proveFinality[hash.H256, uint64, runtime.BlakeTwo256](
		backend,
		authoritySetChanges,
		6,
		true,
	)
	require.NoError(t, err)
	require.Nil(t, proof)
}

func TestFinalityProof_InLastSetUsingLatestJustificationWorks(t *testing.T) {
	// let (client, backend, blocks) = test_blockchain(8, &[4, 5]);
	headerBackend := mocks.NewHeaderBackend[hash.H256, uint64](t)
	backend := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256](t)
	blockchainBackend := mocks.NewBlockchainBackend[hash.H256, uint64](t)
	backend.EXPECT().Blockchain().Return(blockchainBackend)
	blockchainBackend.EXPECT().Info().Return(blockchain.Info[hash.H256, uint64]{
		FinalizedNumber: 8,
	})

	block7 := generic.NewBlock[uint64, hash.H256, runtime.BlakeTwo256](newHeader(7), nil)
	block8 := generic.NewBlock[uint64, hash.H256, runtime.BlakeTwo256](newHeader(8), nil)

	blockchainBackend.EXPECT().ExpectBlockHashFromID(uint64(8)).Return(block8.Hash(), nil)
	blockchainBackend.EXPECT().ExpectBlockHashFromID(uint64(7)).Return(block7.Hash(), nil)

	blockchainBackend.EXPECT().ExpectHeader(block7.Hash()).Return(block7.Header(), nil)
	blockchainBackend.EXPECT().ExpectHeader(block8.Hash()).Return(block8.Header(), nil)

	round := uint64(8)
	commit := createCommit(t, block8, round, 1, []ed25519.Keyring{ed25519.Alice})
	grandpaJust8, err := NewJustificationFromCommit[hash.H256, uint64](headerBackend, round, commit)
	require.NoError(t, err)

	encoded := scale.MustMarshal(grandpaJust8)
	backend.EXPECT().GetAux(bestJustification).Return(&encoded, nil)

	// No recent authority set change, so we are in the authoritySetChangeIDLatest set, and will pickup the best
	// stored justification (via mock get call)
	authoritySetChanges := AuthoritySetChanges[uint64]{}
	authoritySetChanges.append(0, 5)

	proofOf6, err := proveFinality[hash.H256, uint64, runtime.BlakeTwo256](
		backend,
		authoritySetChanges,
		6,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, proofOf6)
	assert.Equal(t, FinalityProof[hash.H256, uint64]{
		Block:          block8.Hash(),
		Justification:  scale.MustMarshal(grandpaJust8),
		UnknownHeaders: []runtime.Header[uint64, hash.H256]{block7.Header(), block8.Header()},
	}, *proofOf6)
}
