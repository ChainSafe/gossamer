// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package warpsync

import (
	"errors"
	"log"
	"math/rand"
	"slices"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/yaml.v3"

	_ "embed"
)

//go:embed testdata/warp_sync_proofs.yaml
var rawWarpSyncProofs []byte

type WarpSyncProofs struct {
	SubstrateWarpSyncProof1 string `yaml:"substrate_warp_sync_proof_1"`
}

func TestDecodeWarpSyncProof(t *testing.T) {
	warpSyncProofs := &WarpSyncProofs{}
	err := yaml.Unmarshal(rawWarpSyncProofs, warpSyncProofs)
	require.NoError(t, err)

	// Generated using substrate
	expected := common.MustHexToBytes(warpSyncProofs.SubstrateWarpSyncProof1)
	if err != nil {
		log.Fatal(err)
	}

	var proof WarpSyncProof

	err = proof.Decode(expected)
	require.NoError(t, err)

	encoded, err := proof.Encode()
	require.NoError(t, err)
	require.Equal(t, expected, encoded)
}
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
	require.Error(t, err)
	require.ErrorIs(t, err, errMissingStartBlock)
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
	require.Error(t, err)
	require.ErrorIs(t, err, errStartBlockNotFinalized)
}

// This test generates a small blockchain with authority set changes and expected
// justifications to create a warp sync proof and verify it.
//
//nolint:lll
func TestGenerateAndVerifyWarpSyncProofOk(t *testing.T) {
	t.Parallel()

	type signedPrecommit = grandpa.SignedPrecommit[hash.H256, uint32, primitives.AuthoritySignature, primitives.AuthorityID]
	type preCommit = grandpa.Precommit[hash.H256, uint32]

	// Initialize mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	blockStateMock := NewMockBlockState(ctrl)
	grandpaStateMock := NewMockGrandpaState(ctrl)

	// Set authorities
	availableAuthorities := ed25519.AvailableAuthorities
	genesisAuthorities := primitives.AuthorityList{
		primitives.AuthorityIDWeight{
			AuthorityID:     ed25519.Alice.Pair().Public().(ced25519.Public),
			AuthorityWeight: 1,
		},
	}
	currentAuthorities := []ed25519.Keyring{ed25519.Alice}

	// Set initial values for the scheduled changes
	currentSetId := primitives.SetID(0)
	authoritySetChanges := []uint{}

	// Genesis block
	genesis := &types.Header{
		ParentHash: common.MustBlake2bHash([]byte("genesis")),
		Number:     1,
	}

	// All blocks headers
	headers := []*types.Header{
		genesis,
	}

	const maxBlocks = 100

	// Create blocks with their scheduled changes and justifications
	for n := uint(1); n <= maxBlocks; n++ {
		lastBlockHeader := headers[len(headers)-1]

		newAuthorities := []ed25519.Keyring{}

		digest := types.NewDigest()

		// Authority set change happens every 10 blocks
		if n != 0 && n%10 == 0 {
			// Pick new random authorities
			nAuthorities := rand.Intn(len(availableAuthorities)-1) + 1
			require.GreaterOrEqual(t, nAuthorities, 1)

			rand.Shuffle(len(availableAuthorities), func(i, j int) {
				availableAuthorities[i], availableAuthorities[j] = availableAuthorities[j], availableAuthorities[i]
			})

			newAuthorities = availableAuthorities[:nAuthorities]

			// Map new authorities to GRANDPA raw authorities format
			nextAuthorities := []types.GrandpaAuthoritiesRaw{}
			for _, key := range newAuthorities {
				nextAuthorities = append(nextAuthorities,
					types.GrandpaAuthoritiesRaw{
						Key: [32]byte(key.Pair().Public().Bytes()),
						ID:  1,
					},
				)
			}

			// Create scheduled change
			scheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
				Auths: nextAuthorities,
				Delay: 0,
			})
			digest.Add(scheduledChange)
		}

		// Create new block header
		header := &types.Header{
			ParentHash: lastBlockHeader.Hash(),
			Number:     lastBlockHeader.Number + 1,
			Digest:     digest,
		}

		headers = append(headers, header)

		// If we have an authority set change, create a justification
		if len(newAuthorities) > 0 {
			targetHash := hash.H256(string(header.Hash().ToBytes()))
			targetNumber := uint32(header.Number)

			// Create precommits for current voters
			precommits := []signedPrecommit{}
			for _, voter := range currentAuthorities {
				precommit := preCommit{
					TargetHash:   targetHash,
					TargetNumber: targetNumber,
				}

				msg := grandpa.NewMessage[hash.H256, uint32, preCommit](precommit)
				encoded := primitives.NewLocalizedPayload(1, currentSetId, msg)
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

			// Create justification
			justification := primitives.GrandpaJustification[hash.H256, uint32]{
				Round: 1,
				Commit: primitives.Commit[hash.H256, uint32]{
					TargetHash:   targetHash,
					TargetNumber: targetNumber,
					Precommits:   precommits,
				},
				VoteAncestries: genericHeadersList(t, headers),
			}

			encodedJustification, err := scale.Marshal(justification)
			require.NoError(t, err)

			blockStateMock.EXPECT().GetJustification(header.Hash()).Return(encodedJustification, nil).AnyTimes()
			blockStateMock.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()

			// Update authorities and set id
			authoritySetChanges = append(authoritySetChanges, header.Number)
			currentAuthorities = slices.Clone(newAuthorities)
			currentSetId++
		}

	}

	// Return expected authority changes for each block
	authChanges := []uint{}
	for n := uint(1); n <= maxBlocks; n++ {
		for _, change := range authoritySetChanges {
			if n <= change {
				authChanges = append(authChanges, change)
			}
		}
		grandpaStateMock.EXPECT().GetAuthoritiesChangesFromBlock(n).Return(authChanges, nil).AnyTimes()
	}

	// Mock responses
	for _, header := range headers {
		blockStateMock.EXPECT().GetHeaderByNumber(header.Number).Return(header, nil).AnyTimes()
		blockStateMock.EXPECT().GetHeader(header.Hash()).Return(header, nil).AnyTimes()
	}

	// Initialize warp sync provider
	provider := NewWarpSyncProofProvider(blockStateMock, grandpaStateMock)

	// Generate proof
	proof, err := provider.Generate(headers[0].Hash())
	require.NoError(t, err)

	// Verify proof
	expectedAuthorities := primitives.AuthorityList{}
	for _, key := range currentAuthorities {
		expectedAuthorities = append(expectedAuthorities,
			primitives.AuthorityIDWeight{
				AuthorityID:     [32]byte(key.Pair().Public().Bytes()),
				AuthorityWeight: 1,
			},
		)
	}

	result, err := provider.Verify(proof, 0, genesisAuthorities)
	require.NoError(t, err)
	require.Equal(t, currentSetId, result.SetId)
	require.Equal(t, expectedAuthorities, result.AuthorityList)
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
	require.NoError(t, err)
	require.NotNil(t, scheduledChangeDigest)
}

func createGRANDPAConsensusDigest(t *testing.T, digestData any) types.ConsensusDigest {
	t.Helper()

	grandpaConsensusDigest := types.NewGrandpaConsensusDigest()
	require.NoError(t, grandpaConsensusDigest.SetValue(digestData))

	marshaledData, err := scale.Marshal(grandpaConsensusDigest)
	require.NoError(t, err)

	return types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              marshaledData,
	}
}

func genericHeadersList(t *testing.T, headers []*types.Header) []runtime.Header[uint32, hash.H256] {
	t.Helper()

	headerList := []runtime.Header[uint32, hash.H256]{}
	for _, header := range headers {
		if header == nil {
			continue
		}

		generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
			uint64(header.Number),
			hash.H256(string(header.ExtrinsicsRoot.ToBytes())),
			hash.H256(string(header.StateRoot.ToBytes())),
			hash.H256(string(header.ParentHash.ToBytes())),
			mapDigest(t, header.Digest),
		)
	}

	return headerList
}

func mapDigest(t *testing.T, digest types.Digest) runtime.Digest {
	newDigest := runtime.Digest{}
	for _, log := range digest {
		value, err := log.Value()
		require.NoError(t, err)

		switch v := value.(type) {
		case types.PreRuntimeDigest:
			newDigest.Push(runtime.NewDigestItem(runtime.PreRuntime{
				ConsensusEngineID: runtime.ConsensusEngineID(v.ConsensusEngineID),
				Bytes:             v.Data,
			}))
		case types.ConsensusDigest:
			newDigest.Push(runtime.NewDigestItem(runtime.Consensus{
				ConsensusEngineID: runtime.ConsensusEngineID(v.ConsensusEngineID),
				Bytes:             v.Data,
			}))
		case types.SealDigest:
			newDigest.Push(runtime.NewDigestItem(runtime.Seal{
				ConsensusEngineID: runtime.ConsensusEngineID(v.ConsensusEngineID),
				Bytes:             v.Data,
			}))
		case types.RuntimeEnvironmentUpdated:
			newDigest.Push(runtime.NewDigestItem(runtime.RuntimeEnvironmentUpdated{}))
		}
	}

	return newDigest
}
