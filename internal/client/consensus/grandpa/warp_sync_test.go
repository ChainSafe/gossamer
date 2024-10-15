// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/lib/common"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGenerateWarpSyncProofBlockNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock block state to return not found block
	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().GetHeader(common.EmptyHash).Return(nil, errors.New("not found")).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState: blockStateMock,
	}

	// Check errMissingStartBlock returned by provider
	_, err := provider.Generate(common.EmptyHash)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errMissingStartBlock)
}

func TestGenerateWarpSyncProofBlockNotFinalized(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock block state to return not found block
	bestBlockHeader := &types.Header{
		Number:     2,
		ParentHash: common.MustBlake2bHash([]byte("1")),
	}

	notFinalizedBlockHeader := &types.Header{
		Number:     3,
		ParentHash: common.MustBlake2bHash([]byte("2")),
	}

	blockStateMock := NewMockBlockState(ctrl)
	blockStateMock.EXPECT().GetHeader(notFinalizedBlockHeader.Hash()).Return(notFinalizedBlockHeader, nil).AnyTimes()
	blockStateMock.EXPECT().GetHighestFinalisedHeader().Return(bestBlockHeader, nil).AnyTimes()

	provider := &WarpSyncProofProvider{
		blockState: blockStateMock,
	}

	// Check errMissingStartBlock returned by provider
	_, err := provider.Generate(notFinalizedBlockHeader.Hash())
	assert.Error(t, err)
	assert.ErrorIs(t, err, errStartBlockNotFinalized)
}

//nolint:lll
func TestGenerateAndVerifyWarpSyncProofOk(t *testing.T) {
	t.Parallel()

	type signedPrecommit = grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	type preCommit = grandpa.Precommit[hash.H256, uint64]

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	blockStateMock := NewMockBlockState(ctrl)
	grandpaStateMock := NewMockGrandpaState(ctrl)

	availableAuthorities := ed25519.AvailableAuthorities
	genesisAuthorities := primitives.AuthorityList{
		primitives.AuthorityIDWeight{
			AuthorityID:     ed25519.Alice.Pair().Public().(ced25519.Public),
			AuthorityWeight: 1,
		},
	}

	currentAuthorities := []ed25519.Keyring{ed25519.Alice}
	currentSetId := primitives.SetID(0)
	authoritySetChanges := []uint{}

	lastBlockHeader := &types.Header{
		ParentHash: common.MustBlake2bHash([]byte("genesis")),
		Number:     1,
	}

	headers := []*types.Header{
		lastBlockHeader,
	}

	const maxBlocks = 2

	for n := uint(1); n <= maxBlocks; n++ {
		newAuthorities := []ed25519.Keyring{}

		digest := types.NewDigest()

		// Authority set change happens every 10 blocks
		if n != 0 && n%2 == 0 {
			nAuthorities := rand.Intn(len(availableAuthorities))
			rand.Shuffle(len(availableAuthorities), func(i, j int) {
				availableAuthorities[i], availableAuthorities[j] = availableAuthorities[j], availableAuthorities[i]
			})

			selectedAuthorities := availableAuthorities[:nAuthorities]
			newAuthorities = selectedAuthorities

			nextAuthorities := []types.GrandpaAuthoritiesRaw{}

			for _, key := range selectedAuthorities {
				nextAuthorities = append(nextAuthorities,
					types.GrandpaAuthoritiesRaw{
						Key: [32]byte(key.Pair().Public().Bytes()),
						ID:  1,
					},
				)
			}

			scheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
				Auths: nextAuthorities,
				Delay: 0,
			})

			digest.Add(scheduledChange)
		}

		header := &types.Header{
			ParentHash: lastBlockHeader.Hash(),
			Number:     lastBlockHeader.Number + 1,
			Digest:     digest,
		}

		headers = append(headers, header)

		lastBlockHeader = header

		if len(newAuthorities) > 0 {
			targetHash := hash.H256(string(header.Hash().ToBytes()))
			targetNumber := uint64(header.Number)

			precommits := []signedPrecommit{}

			for _, voter := range currentAuthorities {
				precommit := preCommit{
					TargetHash:   targetHash,
					TargetNumber: targetNumber,
				}

				msg := grandpa.NewMessage[hash.H256, uint64, preCommit](precommit)
				encoded := primitives.NewLocalizedPayload(1, primitives.SetID(currentSetId), msg)
				signature := voter.Sign(encoded)

				signedPreCommit := signedPrecommit{
					Precommit: preCommit{
						TargetHash:   targetHash,
						TargetNumber: targetNumber,
					},
					Signature: signature,
					ID:        voter.Pair().Public().(ced25519.Public),
				}

				precommits = append(precommits, signedPreCommit)
			}

			justification := primitives.GrandpaJustification[hash.H256, uint64]{
				Round: 1,
				Commit: primitives.Commit[hash.H256, uint64]{
					TargetHash:   targetHash,
					TargetNumber: targetNumber,
					Precommits:   precommits,
				},
				VoteAncestries: genericHeadersList(t, headers),
			}

			encodedJustification, err := scale.Marshal(justification)
			require.NoError(t, err)

			decodedJustification, err := decodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encodedJustification)
			require.NoError(t, err)
			require.Equal(t, justification, decodedJustification.Justification)

			blockStateMock.EXPECT().GetJustification(header.Hash()).Return(encodedJustification, nil).AnyTimes()
			blockStateMock.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()

			authoritySetChanges = append(authoritySetChanges, n)
			currentSetId++
			currentAuthorities = newAuthorities
		}

	}

	authChanges := []uint{}
	for n := uint(1); n <= maxBlocks; n++ {
		for _, change := range authoritySetChanges {
			if n <= change {
				authChanges = append(authChanges, change)
			}
		}
		grandpaStateMock.EXPECT().GetAuthoritiesChangesFromBlock(n).Return(authChanges, nil).AnyTimes()
	}

	for _, header := range headers {
		blockStateMock.EXPECT().GetHeaderByNumber(header.Number).Return(header, nil).AnyTimes()
		blockStateMock.EXPECT().GetHeader(header.Hash()).Return(header, nil).AnyTimes()
	}

	provider := &WarpSyncProofProvider{
		blockState:   blockStateMock,
		grandpaState: grandpaStateMock,
	}

	// Generate proof
	proof, err := provider.Generate(headers[0].Hash())
	assert.NoError(t, err)

	// Verify proof
	result, err := provider.Verify(proof, 0, genesisAuthorities)
	assert.NoError(t, err)
	assert.Equal(t, currentSetId, result.SetId)

	expectedAuthorities := primitives.AuthorityList{}

	for _, key := range currentAuthorities {
		expectedAuthorities = append(expectedAuthorities,
			primitives.AuthorityIDWeight{
				AuthorityID:     [32]byte(key.Pair().Public().Bytes()),
				AuthorityWeight: 1,
			},
		)
	}

	assert.Equal(t, expectedAuthorities, result.AuthorityList)
}

func TestFindScheduledChange(t *testing.T) {
	t.Parallel()

	scheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
		Auths: []types.GrandpaAuthoritiesRaw{},
		Delay: 2,
	})

	digest := types.NewDigest()
	digest.Add(scheduledChange)

	blockHeader := &types.Header{
		ParentHash: common.Hash{0x00},
		Number:     1,
		Digest:     digest,
	}

	// Find scheduled change in block header
	scheduledChangeDigest, err := findScheduledChange(*blockHeader)
	assert.NoError(t, err)
	assert.NotNil(t, scheduledChangeDigest)
}

func createGRANDPAConsensusDigest(t *testing.T, digestData any) types.ConsensusDigest {
	t.Helper()

	grandpaConsensusDigest := types.NewGrandpaConsensusDigest()
	assert.NoError(t, grandpaConsensusDigest.SetValue(digestData))

	marshaledData, err := scale.Marshal(grandpaConsensusDigest)
	assert.NoError(t, err)

	return types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              marshaledData,
	}
}

func genericHeadersList(t *testing.T, headers []*types.Header) []runtime.Header[uint64, hash.H256] {
	t.Helper()

	headerList := []runtime.Header[uint64, hash.H256]{}
	for _, header := range headers {
		if header == nil {
			continue
		}
		newHeader := generic.Header[uint64, hash.H256, runtime.BlakeTwo256]{}
		newHeader.SetParentHash(hash.H256(header.ParentHash.String()))
		newHeader.SetNumber(uint64(header.Number))
		newHeader.DigestMut().Push(header.Digest)
	}

	return headerList
}
