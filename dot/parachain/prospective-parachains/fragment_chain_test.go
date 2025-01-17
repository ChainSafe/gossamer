package prospectiveparachains

import (
	"bytes"
	"errors"
	"maps"
	"math/rand"
	"slices"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
)

func TestCandidateStorage_RemoveCandidate(t *testing.T) {
	storage := &candidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
	}

	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
	parentHeadHash := common.Hash{4, 5, 6}
	outputHeadHash := common.Hash{7, 8, 9}

	entry := &candidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parentHeadHash,
		outputHeadDataHash: outputHeadHash,
		state:              backed,
	}

	storage.byCandidateHash[candidateHash] = entry
	storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}
	storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}

	storage.removeCandidate(candidateHash)

	_, exists := storage.byCandidateHash[candidateHash]
	assert.False(t, exists, "candidate should be removed from byCandidateHash")

	_, exists = storage.byParentHead[parentHeadHash]
	assert.False(t, exists, "candidate should be removed from byParentHead")

	_, exists = storage.byOutputHead[outputHeadHash]
	assert.False(t, exists, "candidate should be removed from byOutputHead")
}

func TestCandidateStorage_MarkBacked(t *testing.T) {
	storage := &candidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
	}

	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
	parentHeadHash := common.Hash{4, 5, 6}
	outputHeadHash := common.Hash{7, 8, 9}

	entry := &candidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parentHeadHash,
		outputHeadDataHash: outputHeadHash,
		state:              seconded,
	}

	storage.byCandidateHash[candidateHash] = entry
	storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}
	storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}

	storage.markBacked(candidateHash)

	assert.Equal(t, backed, entry.state, "candidate state should be marked as backed")
}

func TestCandidateStorage_HeadDataByHash(t *testing.T) {
	tests := map[string]struct {
		setup    func() *candidateStorage
		hash     common.Hash
		expected *parachaintypes.HeadData
	}{
		"find_head_data_of_first_candidate_using_output_head_data_hash": {
			setup: func() *candidateStorage {
				storage := &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}

				candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
				parentHeadHash := common.Hash{4, 5, 6}
				outputHeadHash := common.Hash{7, 8, 9}
				headData := parachaintypes.HeadData{Data: []byte{10, 11, 12}}

				entry := &candidateEntry{
					candidateHash:      candidateHash,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash,
					candidate: &prospectiveCandidate{
						Commitments: parachaintypes.CandidateCommitments{
							HeadData: headData,
						},
					},
				}

				storage.byCandidateHash[candidateHash] = entry
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}
				storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}

				return storage
			},
			hash:     common.Hash{7, 8, 9},
			expected: &parachaintypes.HeadData{Data: []byte{10, 11, 12}},
		},
		"find_head_data_using_parent_head_data_hash_from_second_candidate": {
			setup: func() *candidateStorage {
				storage := &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}

				candidateHash := parachaintypes.CandidateHash{Value: common.Hash{13, 14, 15}}
				parentHeadHash := common.Hash{16, 17, 18}
				outputHeadHash := common.Hash{19, 20, 21}
				headData := parachaintypes.HeadData{Data: []byte{22, 23, 24}}

				entry := &candidateEntry{
					candidateHash:      candidateHash,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash,
					candidate: &prospectiveCandidate{
						PersistedValidationData: parachaintypes.PersistedValidationData{
							ParentHead: headData,
						},
					},
				}

				storage.byCandidateHash[candidateHash] = entry
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}
				storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]struct{}{candidateHash: {}}

				return storage
			},
			hash:     common.Hash{16, 17, 18},
			expected: &parachaintypes.HeadData{Data: []byte{22, 23, 24}},
		},
		"use_nonexistent_hash_and_should_get_nil": {
			setup: func() *candidateStorage {
				storage := &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}
				return storage
			},
			hash:     common.Hash{99, 99, 99},
			expected: nil,
		},
		"insert_0_candidates_and_try_to_find_but_should_get_nil": {
			setup: func() *candidateStorage {
				return &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}
			},
			hash:     common.Hash{7, 8, 9},
			expected: nil,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			storage := tt.setup()
			result := storage.headDataByHash(tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCandidateStorage_PossibleBackedParaChildren(t *testing.T) {
	tests := map[string]struct {
		setup    func() *candidateStorage
		hash     common.Hash
		expected []*candidateEntry
	}{
		"insert_2_candidates_for_same_parent_one_seconded_one_backed": {
			setup: func() *candidateStorage {
				storage := &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}

				candidateHash1 := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
				parentHeadHash := common.Hash{4, 5, 6}
				outputHeadHash1 := common.Hash{7, 8, 9}

				candidateHash2 := parachaintypes.CandidateHash{Value: common.Hash{10, 11, 12}}
				outputHeadHash2 := common.Hash{13, 14, 15}

				entry1 := &candidateEntry{
					candidateHash:      candidateHash1,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash1,
					state:              seconded,
				}

				entry2 := &candidateEntry{
					candidateHash:      candidateHash2,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash2,
					state:              backed,
				}

				storage.byCandidateHash[candidateHash1] = entry1
				storage.byCandidateHash[candidateHash2] = entry2
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]struct{}{
					candidateHash1: {},
					candidateHash2: {},
				}

				return storage
			},
			hash: common.Hash{4, 5, 6},
			expected: []*candidateEntry{{candidateHash: parachaintypes.CandidateHash{
				Value: common.Hash{10, 11, 12}},
				parentHeadDataHash: common.Hash{4, 5, 6},
				outputHeadDataHash: common.Hash{13, 14, 15}, state: backed},
			},
		},
		"insert_nothing_and_call_function_should_return_nothing": {
			setup: func() *candidateStorage {
				return &candidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
				}
			},
			hash:     common.Hash{4, 5, 6},
			expected: nil,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			storage := tt.setup()
			var result []*candidateEntry
			for entry := range storage.possibleBackedParaChildren(tt.hash) {
				result = append(result, entry)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEarliestRelayParent(t *testing.T) {
	tests := map[string]struct {
		setup  func() *scope
		expect relayChainBlockInfo
	}{
		"returns_from_ancestors": {
			setup: func() *scope {
				relayParent := relayChainBlockInfo{
					Hash:   common.Hash{0x01},
					Number: 10,
				}
				baseConstraints := &parachaintypes.Constraints{
					MinRelayParentNumber: 5,
				}
				ancestor := relayChainBlockInfo{
					Hash:   common.Hash{0x02},
					Number: 9,
				}
				ancestorsMap := btree.NewMap[parachaintypes.BlockNumber, relayChainBlockInfo](100)
				ancestorsMap.Set(ancestor.Number, ancestor)
				return &scope{
					relayParent:     relayParent,
					baseConstraints: baseConstraints,
					ancestors:       ancestorsMap,
				}
			},
			expect: relayChainBlockInfo{
				Hash:   common.Hash{0x02},
				Number: 9,
			},
		},
		"returns_relayParent": {
			setup: func() *scope {
				relayParent := relayChainBlockInfo{
					Hash:   common.Hash{0x01},
					Number: 10,
				}
				baseConstraints := &parachaintypes.Constraints{
					MinRelayParentNumber: 5,
				}
				return &scope{
					relayParent:     relayParent,
					baseConstraints: baseConstraints,
					ancestors:       btree.NewMap[parachaintypes.BlockNumber, relayChainBlockInfo](100),
				}
			},
			expect: relayChainBlockInfo{
				Hash:   common.Hash{0x01},
				Number: 10,
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			scope := tt.setup()
			result := scope.earliestRelayParent()
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBackedChain_RevertToParentHash(t *testing.T) {
	tests := map[string]struct {
		setup                    func() *backedChain
		hash                     common.Hash
		expectedChainSize        int
		expectedRemovedFragments int
	}{
		"revert_to_parent_at_pos_2": {
			setup: func() *backedChain {
				chain := &backedChain{
					chain:        make([]*fragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 5; i++ {
					node := &fragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &constraintModifications{},
					}
					chain.push(node)
				}
				return chain
			},
			hash:                     common.Hash{3},
			expectedChainSize:        3,
			expectedRemovedFragments: 2,
		},
		"revert_to_parent_at_pos_0": {
			setup: func() *backedChain {
				chain := &backedChain{
					chain:        make([]*fragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 2; i++ {
					node := &fragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &constraintModifications{},
					}
					chain.push(node)
				}
				return chain
			},
			hash:                     common.Hash{1},
			expectedChainSize:        1,
			expectedRemovedFragments: 1,
		},
		"no_node_removed": {
			setup: func() *backedChain {
				chain := &backedChain{
					chain:        make([]*fragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 3; i++ {
					node := &fragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &constraintModifications{},
					}
					chain.push(node)
				}
				return chain
			},
			hash:                     common.Hash{99}, // Non-existent hash
			expectedChainSize:        3,
			expectedRemovedFragments: 0,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			backedChain := tt.setup()
			removedNodes := backedChain.revertToParentHash(tt.hash)

			// Check the number of removed nodes
			assert.Equal(t, tt.expectedRemovedFragments, len(removedNodes))

			// Check the properties of the chain
			assert.Equal(t, tt.expectedChainSize, len(backedChain.chain))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.byParentHead))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.byOutputHead))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.candidates))

			// Check that the remaining nodes are correct
			for i := 0; i < len(backedChain.chain); i++ {
				assert.Contains(t, backedChain.byParentHead, common.Hash{byte(i)})
				assert.Contains(t, backedChain.byOutputHead, common.Hash{byte(i + 1)})
				assert.Contains(t, backedChain.candidates, parachaintypes.CandidateHash{Value: common.Hash{byte(i)}})
			}
		})
	}
}

func TestFragmentChainWithFreshScope(t *testing.T) {
	relayParent := relayChainBlockInfo{
		Hash:        common.Hash{0x00},
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &parachaintypes.Constraints{
		RequiredParent:       parachaintypes.HeadData{Data: []byte{byte(0)}},
		MinRelayParentNumber: 0,
		ValidationCodeHash:   parachaintypes.ValidationCodeHash(common.Hash{0x03}),
	}

	scope, err := newScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	candidateStorage := newCandidateStorage()

	// Create 3 candidate entries forming a chain
	for i := 0; i < 3; i++ {
		candidateHash := parachaintypes.CandidateHash{Value: [32]byte{byte(i + 1)}}
		parentHead := parachaintypes.HeadData{Data: []byte{byte(i)}}
		outputHead := parachaintypes.HeadData{Data: []byte{byte(i + 1)}}

		persistedValidationData := parachaintypes.PersistedValidationData{
			ParentHead: parentHead,
		}

		// Marshal and hash the persisted validation data
		pvdBytes, err := scale.Marshal(persistedValidationData)
		assert.NoError(t, err)
		pvdHash, err := common.Blake2bHash(pvdBytes)
		assert.NoError(t, err)

		committedCandidate := parachaintypes.CommittedCandidateReceipt{
			Descriptor: parachaintypes.CandidateDescriptor{
				RelayParent:                 common.Hash{0x00},
				PersistedValidationDataHash: pvdHash,
				PovHash:                     common.Hash{0x02},
				ValidationCodeHash:          parachaintypes.ValidationCodeHash(common.Hash{0x03}),
			},
			Commitments: parachaintypes.CandidateCommitments{
				HeadData: outputHead,
			},
		}

		err = candidateStorage.addPendingAvailabilityCandidate(candidateHash, committedCandidate, persistedValidationData)
		assert.NoError(t, err)
	}

	fragmentChain := newFragmentChain(scope, candidateStorage)

	// Check that the best chain contains 3 candidates
	assert.Equal(t, 3, len(fragmentChain.bestChain.chain))
}

func makeConstraints(
	minRelayParentNumber parachaintypes.BlockNumber,
	validWatermarks []parachaintypes.BlockNumber,
	requiredParent parachaintypes.HeadData,
) *parachaintypes.Constraints {
	return &parachaintypes.Constraints{
		MinRelayParentNumber:  minRelayParentNumber,
		MaxPoVSize:            1_000_000,
		MaxCodeSize:           1_000_000,
		UMPRemaining:          10,
		UMPRemainingBytes:     1_000,
		MaxNumUMPPerCandidate: 10,
		DMPRemainingMessages:  make([]parachaintypes.BlockNumber, 10),
		HRMPInbound: parachaintypes.InboundHRMPLimitations{
			ValidWatermarks: validWatermarks,
		},
		HRMPChannelsOut:        make(map[parachaintypes.ParaID]parachaintypes.OutboundHRMPChannelLimitations),
		MaxNumHRMPPerCandidate: 0,
		RequiredParent:         requiredParent,
		ValidationCodeHash:     parachaintypes.ValidationCodeHash(common.BytesToHash(bytes.Repeat([]byte{42}, 32))),
		UpgradeRestriction:     nil,
		FutureValidationCode:   nil,
	}
}

func makeCommittedCandidate(
	t *testing.T,
	paraID parachaintypes.ParaID,
	relayParent common.Hash,
	relayParentNumber uint32,
	parentHead parachaintypes.HeadData,
	paraHead parachaintypes.HeadData,
	hrmpWatermark uint32,
) (parachaintypes.PersistedValidationData, parachaintypes.CommittedCandidateReceipt) {
	persistedValidationData := parachaintypes.PersistedValidationData{
		ParentHead:             parentHead,
		RelayParentNumber:      relayParentNumber,
		RelayParentStorageRoot: common.Hash{},
		MaxPovSize:             1_000_000,
	}

	pvdBytes, err := scale.Marshal(persistedValidationData)
	require.NoError(t, err)

	pvdHash, err := common.Blake2bHash(pvdBytes)
	require.NoError(t, err)

	paraHeadHash, err := paraHead.Hash()
	require.NoError(t, err)

	candidate := parachaintypes.CommittedCandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      paraID,
			RelayParent:                 relayParent,
			Collator:                    parachaintypes.CollatorID([sr25519.PublicKeyLength]byte{}),
			PersistedValidationDataHash: pvdHash,
			PovHash:                     common.BytesToHash(bytes.Repeat([]byte{1}, 32)),
			ErasureRoot:                 common.BytesToHash(bytes.Repeat([]byte{1}, 32)),
			Signature:                   parachaintypes.CollatorSignature([sr25519.SignatureLength]byte{}),
			ParaHead:                    paraHeadHash,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(common.BytesToHash(bytes.Repeat([]byte{42}, 32))),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:            []parachaintypes.UpwardMessage{},
			HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
			NewValidationCode:         nil,
			HeadData:                  paraHead,
			ProcessedDownwardMessages: 1,
			HrmpWatermark:             hrmpWatermark,
		},
	}

	return persistedValidationData, candidate
}

func TestScopeRejectsAncestors(t *testing.T) {
	tests := map[string]struct {
		relayParent         *relayChainBlockInfo
		ancestors           []relayChainBlockInfo
		maxDepth            uint
		baseConstraints     *parachaintypes.Constraints
		pendingAvailability []*pendingAvailability
		expectedError       error
	}{
		"rejects_ancestor_that_skips_blocks": {
			relayParent: &relayChainBlockInfo{
				Number:      10,
				Hash:        common.BytesToHash(bytes.Repeat([]byte{0x10}, 32)),
				StorageRoot: common.BytesToHash(bytes.Repeat([]byte{0x69}, 32)),
			},
			ancestors: []relayChainBlockInfo{
				{
					Number:      8,
					Hash:        common.BytesToHash(bytes.Repeat([]byte{0x08}, 32)),
					StorageRoot: common.BytesToHash(bytes.Repeat([]byte{0x69}, 69)),
				},
			},
			maxDepth: 2,
			baseConstraints: makeConstraints(8, []parachaintypes.BlockNumber{8, 9},
				parachaintypes.HeadData{Data: []byte{0x01, 0x02, 0x03}}),
			pendingAvailability: make([]*pendingAvailability, 0),
			expectedError:       errUnexpectedAncestor{number: 8, prev: 10},
		},
		"rejects_ancestor_for_zero_block": {
			relayParent: &relayChainBlockInfo{
				Number:      0,
				Hash:        common.BytesToHash(bytes.Repeat([]byte{0}, 32)),
				StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
			},
			ancestors: []relayChainBlockInfo{
				{
					Number:      99999,
					Hash:        common.BytesToHash(bytes.Repeat([]byte{99}, 32)),
					StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
				},
			},
			maxDepth: 2,
			baseConstraints: makeConstraints(0, []parachaintypes.BlockNumber{},
				parachaintypes.HeadData{Data: []byte{1, 2, 3}}),
			pendingAvailability: make([]*pendingAvailability, 0),
			expectedError:       errUnexpectedAncestor{number: 99999, prev: 0},
		},
		"rejects_unordered_ancestors": {
			relayParent: &relayChainBlockInfo{
				Number:      5,
				Hash:        common.BytesToHash(bytes.Repeat([]byte{0}, 32)),
				StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
			},
			ancestors: []relayChainBlockInfo{
				{
					Number:      4,
					Hash:        common.BytesToHash(bytes.Repeat([]byte{4}, 32)),
					StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
				},
				{
					Number:      2,
					Hash:        common.BytesToHash(bytes.Repeat([]byte{2}, 32)),
					StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
				},
				{
					Number:      3,
					Hash:        common.BytesToHash(bytes.Repeat([]byte{3}, 32)),
					StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
				},
			},
			maxDepth: 2,
			baseConstraints: makeConstraints(0, []parachaintypes.BlockNumber{2},
				parachaintypes.HeadData{Data: []byte{1, 2, 3}}),
			pendingAvailability: make([]*pendingAvailability, 0),
			expectedError:       errUnexpectedAncestor{number: 2, prev: 4},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			scope, err := newScopeWithAncestors(
				*tt.relayParent,
				tt.baseConstraints,
				tt.pendingAvailability,
				tt.maxDepth,
				tt.ancestors)
			require.ErrorIs(t, err, tt.expectedError)
			require.Nil(t, scope)
		})
	}
}

func TestScopeOnlyTakesAncestorsUpToMin(t *testing.T) {
	relayParent := relayChainBlockInfo{
		Number:      5,
		Hash:        common.BytesToHash(bytes.Repeat([]byte{0}, 32)),
		StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
	}

	ancestors := []relayChainBlockInfo{
		{
			Number:      4,
			Hash:        common.BytesToHash(bytes.Repeat([]byte{4}, 32)),
			StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
		},
		{
			Number:      3,
			Hash:        common.BytesToHash(bytes.Repeat([]byte{3}, 32)),
			StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
		},
		{
			Number:      2,
			Hash:        common.BytesToHash(bytes.Repeat([]byte{2}, 32)),
			StorageRoot: common.BytesToHash(bytes.Repeat([]byte{69}, 32)),
		},
	}

	maxDepth := uint(2)
	baseConstraints := makeConstraints(3, []parachaintypes.BlockNumber{2}, parachaintypes.HeadData{Data: []byte{1, 2, 3}})
	pendingAvailability := make([]*pendingAvailability, 0)

	scope, err := newScopeWithAncestors(relayParent, baseConstraints, pendingAvailability, maxDepth, ancestors)
	require.NoError(t, err)

	assert.Equal(t, 2, scope.ancestors.Len())
	assert.Equal(t, 2, len(scope.ancestorsByHash))
}

func TestCandidateStorageMethods(t *testing.T) {
	tests := map[string]struct {
		runTest func(*testing.T)
	}{
		"persistedValidationDataMismatch": {
			runTest: func(t *testing.T) {
				relayParent := common.BytesToHash(bytes.Repeat([]byte{69}, 32))

				pvd, candidate := makeCommittedCandidate(
					t,
					parachaintypes.ParaID(5),
					relayParent,
					8,
					parachaintypes.HeadData{Data: []byte{4, 5, 6}},
					parachaintypes.HeadData{Data: []byte{1, 2, 3}},
					7,
				)

				wrongPvd := pvd
				wrongPvd.MaxPovSize = 0

				candidateHash, err := candidate.Hash()
				require.NoError(t, err)

				entry, err := newCandidateEntry(parachaintypes.CandidateHash{Value: candidateHash},
					candidate, wrongPvd, seconded)
				require.ErrorIs(t, err, errPersistedValidationDataMismatch)
				require.Nil(t, entry)
			},
		},

		"zero_length_cycle": {
			runTest: func(t *testing.T) {
				relayParent := common.BytesToHash(bytes.Repeat([]byte{69}, 32))

				pvd, candidate := makeCommittedCandidate(
					t,
					parachaintypes.ParaID(5),
					relayParent,
					8,
					parachaintypes.HeadData{Data: []byte{4, 5, 6}},
					parachaintypes.HeadData{Data: []byte{1, 2, 3}},
					7,
				)

				candidate.Commitments.HeadData = parachaintypes.HeadData{Data: bytes.Repeat([]byte{1}, 10)}
				pvd.ParentHead = parachaintypes.HeadData{Data: bytes.Repeat([]byte{1}, 10)}
				wrongPvdHash, err := pvd.Hash()
				require.NoError(t, err)

				candidate.Descriptor.PersistedValidationDataHash = wrongPvdHash

				candidateHash, err := candidate.Hash()
				require.NoError(t, err)

				entry, err := newCandidateEntry(parachaintypes.CandidateHash{Value: candidateHash},
					candidate, pvd, seconded)
				require.Nil(t, entry)
				require.ErrorIs(t, err, errZeroLengthCycle)
			},
		},

		"add_valid_candidate": {
			runTest: func(t *testing.T) {
				relayParent := common.BytesToHash(bytes.Repeat([]byte{69}, 32))

				pvd, candidate := makeCommittedCandidate(
					t,
					parachaintypes.ParaID(5),
					relayParent,
					8,
					parachaintypes.HeadData{Data: []byte{4, 5, 6}},
					parachaintypes.HeadData{Data: []byte{1, 2, 3}},
					7,
				)

				hash, err := candidate.Hash()
				require.NoError(t, err)
				candidateHash := parachaintypes.CandidateHash{Value: hash}

				parentHeadHash, err := pvd.ParentHead.Hash()
				require.NoError(t, err)

				entry, err := newCandidateEntry(candidateHash, candidate, pvd, seconded)
				require.NoError(t, err)

				storage := newCandidateStorage()

				t.Run("add_candidate_entry_as_seconded", func(t *testing.T) {
					err = storage.addCandidateEntry(entry)
					require.NoError(t, err)
					_, ok := storage.byCandidateHash[candidateHash]
					require.True(t, ok)

					// should not have any possible backed candidate yet
					for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
						assert.Fail(t, "expected no entries, but found one", entry)
					}

					require.Equal(t, storage.headDataByHash(candidate.Descriptor.ParaHead),
						&candidate.Commitments.HeadData)
					require.Equal(t, storage.headDataByHash(parentHeadHash), &pvd.ParentHead)

					// re-add the candidate should fail
					err = storage.addCandidateEntry(entry)
					require.ErrorIs(t, err, errCandidateAlreadyKnown)
				})

				t.Run("mark_candidate_entry_as_backed", func(t *testing.T) {
					storage.markBacked(candidateHash)
					// marking twice is fine
					storage.markBacked(candidateHash)

					// here we should have 1 possible backed candidate when we
					// use the parentHeadHash (parent of our current candidate) to query
					possibleBackedCandidateHashes := make([]parachaintypes.CandidateHash, 0)
					for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
						possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
					}

					require.Equal(t, []parachaintypes.CandidateHash{candidateHash}, possibleBackedCandidateHashes)

					// here we should have 0 possible backed candidate because we are
					// using the candidate hash paraHead as base to query
					possibleBackedCandidateHashes = make([]parachaintypes.CandidateHash, 0)
					for entry := range storage.possibleBackedParaChildren(candidate.Descriptor.ParaHead) {
						possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
					}

					require.Empty(t, possibleBackedCandidateHashes)
				})

				t.Run("remove_candidate_entry", func(t *testing.T) {
					storage.removeCandidate(candidateHash)
					// remove it twice should be fine
					storage.removeCandidate(candidateHash)

					_, ok := storage.byCandidateHash[candidateHash]
					require.False(t, ok)

					// should not have any possible backed candidate anymore
					for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
						assert.Fail(t, "expected no entries, but found one", entry)
					}

					require.Nil(t, storage.headDataByHash(candidate.Descriptor.ParaHead))
					require.Nil(t, storage.headDataByHash(parentHeadHash))
				})
			},
		},

		"add_pending_availability_candidate": {
			runTest: func(t *testing.T) {
				relayParent := common.BytesToHash(bytes.Repeat([]byte{69}, 32))

				pvd, candidate := makeCommittedCandidate(
					t,
					parachaintypes.ParaID(5),
					relayParent,
					8,
					parachaintypes.HeadData{Data: []byte{4, 5, 6}},
					parachaintypes.HeadData{Data: []byte{1, 2, 3}},
					7,
				)

				hash, err := candidate.Hash()
				require.NoError(t, err)
				candidateHash := parachaintypes.CandidateHash{Value: hash}

				parentHeadHash, err := pvd.ParentHead.Hash()
				require.NoError(t, err)

				storage := newCandidateStorage()
				err = storage.addPendingAvailabilityCandidate(candidateHash, candidate, pvd)
				require.NoError(t, err)

				_, ok := storage.byCandidateHash[candidateHash]
				require.True(t, ok)

				// here we should have 1 possible backed candidate when we
				// use the parentHeadHash (parent of our current candidate) to query
				possibleBackedCandidateHashes := make([]parachaintypes.CandidateHash, 0)
				for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
					possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
				}

				require.Equal(t, []parachaintypes.CandidateHash{candidateHash}, possibleBackedCandidateHashes)

				// here we should have 0 possible backed candidate because we are
				// using the candidate hash paraHead as base to query
				possibleBackedCandidateHashes = make([]parachaintypes.CandidateHash, 0)
				for entry := range storage.possibleBackedParaChildren(candidate.Descriptor.ParaHead) {
					possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
				}

				require.Empty(t, possibleBackedCandidateHashes)

				t.Run("add_seconded_candidate_to_create_fork", func(t *testing.T) {
					pvd2, candidate2 := makeCommittedCandidate(
						t,
						parachaintypes.ParaID(5),
						relayParent,
						8,
						parachaintypes.HeadData{Data: []byte{4, 5, 6}},
						parachaintypes.HeadData{Data: []byte{2, 3, 4}},
						7,
					)

					hash2, err := candidate2.Hash()
					require.NoError(t, err)
					candidateHash2 := parachaintypes.CandidateHash{Value: hash2}

					candidateEntry2, err := newCandidateEntry(candidateHash2, candidate2, pvd2, seconded)
					require.NoError(t, err)

					err = storage.addCandidateEntry(candidateEntry2)
					require.NoError(t, err)

					// here we should have 1 possible backed candidate since
					// the other candidate is seconded
					possibleBackedCandidateHashes := make([]parachaintypes.CandidateHash, 0)
					for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
						possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
					}

					require.Equal(t, []parachaintypes.CandidateHash{candidateHash}, possibleBackedCandidateHashes)

					// now mark it as backed
					storage.markBacked(candidateHash2)

					// here we should have 1 possible backed candidate since
					// the other candidate is seconded
					possibleBackedCandidateHashes = make([]parachaintypes.CandidateHash, 0)
					for entry := range storage.possibleBackedParaChildren(parentHeadHash) {
						possibleBackedCandidateHashes = append(possibleBackedCandidateHashes, entry.candidateHash)
					}

					require.Equal(t, []parachaintypes.CandidateHash{
						candidateHash, candidateHash2}, possibleBackedCandidateHashes)

				})
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			tt.runTest(t)
		})
	}
}

func TestInitAndPopulateFromEmpty(t *testing.T) {
	baseConstraints := makeConstraints(0, []parachaintypes.BlockNumber{0}, parachaintypes.HeadData{Data: []byte{0x0a}})

	scope, err := newScopeWithAncestors(
		relayChainBlockInfo{
			Number:      1,
			Hash:        common.BytesToHash(bytes.Repeat([]byte{1}, 32)),
			StorageRoot: common.BytesToHash(bytes.Repeat([]byte{2}, 32)),
		},
		baseConstraints,
		nil,
		4,
		nil,
	)
	require.NoError(t, err)

	chain := newFragmentChain(scope, newCandidateStorage())
	assert.Equal(t, 0, chain.bestChainLen())
	assert.Equal(t, 0, chain.unconnected.len())

	newChain := newFragmentChain(scope, newCandidateStorage())
	newChain.populateFromPrevious(chain)
	assert.Equal(t, 0, newChain.bestChainLen())
	assert.Equal(t, 0, newChain.unconnected.len())
}

func populateFromPreviousStorage(scope *scope, storage *candidateStorage) *fragmentChain {
	chain := newFragmentChain(scope, newCandidateStorage())

	// clone the value
	prevChain := *chain
	(&prevChain).unconnected = storage.clone()
	chain.populateFromPrevious(&prevChain)
	return chain
}

func TestPopulateAndCheckPotential(t *testing.T) {
	storage := newCandidateStorage()
	paraID := parachaintypes.ParaID(5)

	relayParentAHash := common.BytesToHash(bytes.Repeat([]byte{1}, 32))
	relayParentBHash := common.BytesToHash(bytes.Repeat([]byte{2}, 32))
	relayParentCHash := common.BytesToHash(bytes.Repeat([]byte{3}, 32))

	relayParentAInfo := &relayChainBlockInfo{
		Number: 0, Hash: relayParentAHash, StorageRoot: common.Hash{},
	}

	relayParentBInfo := &relayChainBlockInfo{
		Number: 1, Hash: relayParentBHash, StorageRoot: common.Hash{},
	}

	relayParentCInfo := &relayChainBlockInfo{
		Number: 2, Hash: relayParentCHash, StorageRoot: common.Hash{},
	}

	// the ancestors must be in the reverse order
	ancestors := []relayChainBlockInfo{
		*relayParentBInfo,
		*relayParentAInfo,
	}

	firstParachainHead := parachaintypes.HeadData{Data: []byte{0x0a}}
	baseConstraints := makeConstraints(0, []parachaintypes.BlockNumber{0}, firstParachainHead)

	// helper function to hash the candidate and add its entry
	// into the candidate storage
	hashAndInsertCandididate := func(t *testing.T, storage *candidateStorage,
		candidate parachaintypes.CommittedCandidateReceipt,
		pvd parachaintypes.PersistedValidationData, state candidateState) (
		parachaintypes.CandidateHash, *candidateEntry) {

		hash, err := candidate.Hash()
		require.NoError(t, err)
		candidateHash := parachaintypes.CandidateHash{Value: hash}
		entry, err := newCandidateEntry(candidateHash, candidate, pvd, state)
		require.NoError(t, err)
		err = storage.addCandidateEntry(entry)
		require.NoError(t, err)

		return candidateHash, entry
	}

	hashAndGetEntry := func(t *testing.T, candidate parachaintypes.CommittedCandidateReceipt,
		pvd parachaintypes.PersistedValidationData, state candidateState) (parachaintypes.CandidateHash, *candidateEntry) {
		hash, err := candidate.Hash()
		require.NoError(t, err)
		candidateHash := parachaintypes.CandidateHash{Value: hash}
		entry, err := newCandidateEntry(candidateHash, candidate, pvd, state)
		require.NoError(t, err)
		return candidateHash, entry
	}

	// candidates A -> B -> C are all backed
	candidateAParaHead := parachaintypes.HeadData{Data: []byte{0x0b}}
	pvdA, candidateA := makeCommittedCandidate(t, paraID,
		relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
		firstParachainHead,
		candidateAParaHead,
		uint32(relayParentAInfo.Number),
	)

	candidateAHash, candidateAEntry := hashAndInsertCandididate(t, storage, candidateA, pvdA, backed)

	candidateBParaHead := parachaintypes.HeadData{Data: []byte{0x0c}}
	pvdB, candidateB := makeCommittedCandidate(t, paraID,
		relayParentBInfo.Hash, uint32(relayParentBInfo.Number),
		candidateAParaHead, // defines candidate A as parent of candidate B
		candidateBParaHead,
		uint32(relayParentBInfo.Number),
	)

	candidateBHash, candidateBEntry := hashAndInsertCandididate(t, storage, candidateB, pvdB, backed)

	candidateCParaHead := parachaintypes.HeadData{Data: []byte{0x0d}}
	pvdC, candidateC := makeCommittedCandidate(t, paraID,
		relayParentCInfo.Hash, uint32(relayParentCInfo.Number),
		candidateBParaHead,
		candidateCParaHead,
		uint32(relayParentCInfo.Number),
	)

	candidateCHash, candidateCEntry := hashAndInsertCandididate(t, storage, candidateC, pvdC, backed)

	t.Run("candidate_A_doesnt_adhere_to_base_constraints", func(t *testing.T) {
		wrongConstraints := []parachaintypes.Constraints{
			// define a constraint that requires a parent head data
			// that is different from candidate A parent head
			*makeConstraints(relayParentAInfo.Number,
				[]parachaintypes.BlockNumber{relayParentAInfo.Number}, parachaintypes.HeadData{Data: []byte{0x0e}}),

			// the min relay parent for candidate A is wrong
			*makeConstraints(relayParentBInfo.Number, []parachaintypes.BlockNumber{0}, firstParachainHead),
		}

		for _, wrongConstraint := range wrongConstraints {
			scope, err := newScopeWithAncestors(
				*relayParentCInfo,
				&wrongConstraint,
				nil,
				4,
				ancestors,
			)
			require.NoError(t, err)

			chain := populateFromPreviousStorage(scope, storage)
			require.Empty(t, chain.bestChainVec())

			// if the min relay parent is wrong, candidate A can never become valid, otherwise
			// if only the required parent doesnt match, candidate A still a potential candidate
			if wrongConstraint.MinRelayParentNumber == relayParentBInfo.Number {
				// if A is not a potential candidate, its descendants will also not be added.
				require.Equal(t, chain.unconnected.len(), 0)
				err := chain.canAddCandidateAsPotential(candidateAEntry)
				require.ErrorIs(t, err, errRelayParentNotInScope{
					relayParentA: relayParentAHash, // candidate A has relay parent A
					relayParentB: relayParentBHash, // while the constraint is expecting at least relay parent B
				})

				// however if taken independently, both B and C still have potential
				err = chain.canAddCandidateAsPotential(candidateBEntry)
				require.NoError(t, err)
				err = chain.canAddCandidateAsPotential(candidateCEntry)
				require.NoError(t, err)
			} else {
				potentials := make([]parachaintypes.CandidateHash, 0)
				for _, unconnected := range chain.unconnected.byCandidateHash {
					potentials = append(potentials, unconnected.candidateHash)
				}

				slices.SortStableFunc(potentials, func(i, j parachaintypes.CandidateHash) int {
					return bytes.Compare(i.Value[:], j.Value[:])
				})

				require.Equal(t, []parachaintypes.CandidateHash{
					candidateAHash,
					candidateCHash,
					candidateBHash,
				}, potentials)
			}
		}
	})

	t.Run("depth_cases", func(t *testing.T) {
		depthCases := map[string]struct {
			depth               []uint
			expectedBestChain   []parachaintypes.CandidateHash
			expectedUnconnected map[parachaintypes.CandidateHash]struct{}
		}{
			"0_depth_only_allows_one_candidate_but_keep_the_rest_as_potential": {
				depth:             []uint{0},
				expectedBestChain: []parachaintypes.CandidateHash{candidateAHash},
				expectedUnconnected: map[parachaintypes.CandidateHash]struct{}{
					candidateBHash: {},
					candidateCHash: {},
				},
			},
			"1_depth_allow_two_candidates": {
				depth:             []uint{1},
				expectedBestChain: []parachaintypes.CandidateHash{candidateAHash, candidateBHash},
				expectedUnconnected: map[parachaintypes.CandidateHash]struct{}{
					candidateCHash: {},
				},
			},
			"2_more_depth_allow_all_candidates": {
				depth:               []uint{2, 3, 4, 5},
				expectedBestChain:   []parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash},
				expectedUnconnected: map[parachaintypes.CandidateHash]struct{}{},
			},
		}

		for tname, tt := range depthCases {
			tt := tt
			t.Run(tname, func(t *testing.T) {
				// iterate over all the depth values
				for _, depth := range tt.depth {
					scope, err := newScopeWithAncestors(
						*relayParentCInfo,
						baseConstraints,
						nil,
						depth,
						ancestors,
					)
					require.NoError(t, err)

					chain := newFragmentChain(scope, newCandidateStorage())
					// individually each candidate is a potential candidate
					require.NoError(t, chain.canAddCandidateAsPotential(candidateAEntry))
					require.NoError(t, chain.canAddCandidateAsPotential(candidateBEntry))
					require.NoError(t, chain.canAddCandidateAsPotential(candidateCEntry))

					chain = populateFromPreviousStorage(scope, storage)
					require.Equal(t, tt.expectedBestChain, chain.bestChainVec())

					// Check that the unconnected candidates are as expected
					unconnectedHashes := make(map[parachaintypes.CandidateHash]struct{})
					for _, unconnected := range chain.unconnected.byCandidateHash {
						unconnectedHashes[unconnected.candidateHash] = struct{}{}
					}

					assert.Equal(t, tt.expectedUnconnected, unconnectedHashes)
				}
			})
		}
	})

	t.Run("relay_parent_out_of_scope", func(t *testing.T) {
		// candidate A has a relay parent out of scope. Candidates B and C
		// will also be deleted since they form a chain with A
		t.Run("candidate_A_relay_parent_out_of_scope", func(t *testing.T) {
			newAncestors := []relayChainBlockInfo{
				*relayParentBInfo,
			}

			scope, err := newScopeWithAncestors(
				*relayParentCInfo,
				baseConstraints,
				nil,
				4,
				newAncestors,
			)
			require.NoError(t, err)
			chain := populateFromPreviousStorage(scope, storage)
			require.Empty(t, chain.bestChainVec())
			require.Equal(t, 0, chain.unconnected.len())

			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateAEntry),
				errRelayParentNotInScope{
					relayParentA: relayParentAHash,
					relayParentB: relayParentBHash,
				})

			// however if taken indepently, both B and C still have potential
			require.NoError(t, chain.canAddCandidateAsPotential(candidateBEntry))
			require.NoError(t, chain.canAddCandidateAsPotential(candidateCEntry))
		})

		t.Run("candidate_A_and_B_out_of_scope_C_still_potential", func(t *testing.T) {
			scope, err := newScopeWithAncestors(
				*relayParentCInfo,
				baseConstraints,
				nil,
				4,
				nil,
			)
			require.NoError(t, err)
			chain := populateFromPreviousStorage(scope, storage)
			require.Empty(t, chain.bestChainVec())
			require.Equal(t, 0, chain.unconnected.len())

			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateAEntry),
				errRelayParentNotInScope{
					relayParentA: relayParentAHash,
					relayParentB: relayParentCHash,
				})

			// however if taken indepently, both B and C still have potential
			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateBEntry),
				errRelayParentNotInScope{
					relayParentA: relayParentBHash,
					relayParentB: relayParentCHash,
				})

			require.NoError(t, chain.canAddCandidateAsPotential(candidateCEntry))
		})
	})

	t.Run("parachain_cycle_not_allowed", func(t *testing.T) {
		// make C parent of parachain block A
		modifiedStorage := storage.clone()
		modifiedStorage.removeCandidate(candidateCHash)

		wrongPvdC, wrongCandidateC := makeCommittedCandidate(t, paraID,
			relayParentCInfo.Hash, uint32(relayParentCInfo.Number),
			candidateBParaHead, // defines candidate B as parent of candidate C
			firstParachainHead, //  defines this candidate para head output as the parent of candidate A
			uint32(relayParentCInfo.Number),
		)

		_, wrongCandidateCEntry := hashAndInsertCandididate(t, modifiedStorage, wrongCandidateC, wrongPvdC, backed)

		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			nil,
			4,
			ancestors,
		)
		require.NoError(t, err)

		chain := populateFromPreviousStorage(scope, modifiedStorage)
		require.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash}, chain.bestChainVec())
		require.Equal(t, 0, chain.unconnected.len())

		err = chain.canAddCandidateAsPotential(wrongCandidateCEntry)
		require.ErrorIs(t, err, errCycle)

		// However, if taken independently, C still has potential, since we don't know A and B.
		chain = newFragmentChain(scope, newCandidateStorage())
		require.NoError(t, chain.canAddCandidateAsPotential(wrongCandidateCEntry))
	})

	t.Run("relay_parent_move_backwards_not_allowed", func(t *testing.T) {
		// each candidate was build using a different, and contigous, relay parent
		// in this test we are going to change candidate C to have the same relay
		// parent of candidate A, given that candidate B is one block ahead.
		modifiedStorage := storage.clone()
		modifiedStorage.removeCandidate(candidateCHash)

		wrongPvdC, wrongCandidateC := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			candidateBParaHead,
			candidateCParaHead,
			0,
		)

		_, wrongCandidateCEntry := hashAndInsertCandididate(t, modifiedStorage, wrongCandidateC, wrongPvdC, backed)

		scope, err := newScopeWithAncestors(*relayParentCInfo, baseConstraints, nil, 4, ancestors)
		require.NoError(t, err)

		chain := populateFromPreviousStorage(scope, modifiedStorage)
		require.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash}, chain.bestChainVec())
		require.Equal(t, 0, chain.unconnected.len())

		require.ErrorIs(t, chain.canAddCandidateAsPotential(wrongCandidateCEntry), errRelayParentMovedBackwards)
	})

	t.Run("unconnected_candidate_C", func(t *testing.T) {
		// candidate C is an unconnected candidate, C's relay parent is allowed to move
		// backwards from B's relay parent, because C may latter on trigger a reorg and
		// B may get removed

		modifiedStorage := storage.clone()
		modifiedStorage.removeCandidate(candidateCHash)

		parenteHead := parachaintypes.HeadData{Data: []byte{0x0d}}
		unconnectedCandidateCHead := parachaintypes.HeadData{Data: []byte{0x0e}}

		unconnectedCPvd, unconnectedCandidateC := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			parenteHead,
			unconnectedCandidateCHead,
			0,
		)

		unconnectedCandidateCHash, unconnectedCandidateCEntry := hashAndInsertCandididate(t,
			modifiedStorage, unconnectedCandidateC, unconnectedCPvd, backed)

		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			nil,
			4,
			ancestors,
		)
		require.NoError(t, err)

		chain := newFragmentChain(scope, newCandidateStorage())
		require.NoError(t, chain.canAddCandidateAsPotential(unconnectedCandidateCEntry))

		chain = populateFromPreviousStorage(scope, modifiedStorage)
		require.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash}, chain.bestChainVec())

		unconnected := make(map[parachaintypes.CandidateHash]struct{})
		for _, entry := range chain.unconnected.byCandidateHash {
			unconnected[entry.candidateHash] = struct{}{}
		}

		require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
			unconnectedCandidateCHash: {},
		}, unconnected)

		t.Run("candidate_A_is_pending_availability_candidate_C_should_not_move_backwards", func(t *testing.T) {
			// candidate A is pending availability and candidate C is an unconnected candidate, C's relay parent
			// is not allowed to move backwards from A's relay parent because we're sure A will not get remove
			// in the future, as it's already on-chain (unless it times out availability, a case for which we
			// don't care to optmise for)
			modifiedStorage.removeCandidate(candidateAHash)
			modifiedAPvd, modifiedCandidateA := makeCommittedCandidate(t, paraID,
				relayParentBInfo.Hash, uint32(relayParentBInfo.Number),
				firstParachainHead,
				candidateAParaHead,
				uint32(relayParentBInfo.Number),
			)

			modifiedCandidateAHash, _ := hashAndInsertCandididate(t,
				modifiedStorage, modifiedCandidateA, modifiedAPvd, backed)

			scope, err := newScopeWithAncestors(
				*relayParentCInfo,
				baseConstraints,
				[]*pendingAvailability{
					{candidateHash: modifiedCandidateAHash, relayParent: *relayParentBInfo},
				},
				4,
				ancestors,
			)
			require.NoError(t, err)

			chain := populateFromPreviousStorage(scope, modifiedStorage)
			require.Equal(t, []parachaintypes.CandidateHash{modifiedCandidateAHash, candidateBHash}, chain.bestChainVec())
			require.Equal(t, 0, chain.unconnected.len())

			require.ErrorIs(t,
				chain.canAddCandidateAsPotential(unconnectedCandidateCEntry),
				errRelayParentPrecedesCandidatePendingAvailability{
					relayParentA: relayParentAHash,
					relayParentB: relayParentBHash,
				})
		})
	})

	t.Run("cannot_fork_from_a_candidate_pending_availability", func(t *testing.T) {
		modifiedStorage := storage.clone()
		modifiedStorage.removeCandidate(candidateCHash)

		modifiedStorage.removeCandidate(candidateAHash)
		modifiedAPvd, modifiedCandidateA := makeCommittedCandidate(t, paraID,
			relayParentBInfo.Hash, uint32(relayParentBInfo.Number),
			firstParachainHead,
			candidateAParaHead,
			uint32(relayParentBInfo.Number),
		)

		modifiedCandidateAHash, _ := hashAndInsertCandididate(t,
			modifiedStorage, modifiedCandidateA, modifiedAPvd, backed)

		wrongCandidateCHead := parachaintypes.HeadData{Data: []byte{0x01}}
		wrongPvdC, wrongCandidateC := makeCommittedCandidate(t, paraID,
			relayParentBInfo.Hash, uint32(relayParentBInfo.Number),
			firstParachainHead,
			wrongCandidateCHead,
			uint32(relayParentBInfo.Number),
		)

		wrongCandidateCHash, wrongCandidateCEntry := hashAndInsertCandididate(t,
			modifiedStorage, wrongCandidateC, wrongPvdC, backed)

		// does not matter if the fork selection rule picks the new candidate
		// as the modified candidate A is pending availability
		require.Equal(t, -1, forkSelectionRule(wrongCandidateCHash, modifiedCandidateAHash))

		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			[]*pendingAvailability{
				{
					candidateHash: modifiedCandidateAHash,
					relayParent:   *relayParentBInfo,
				},
			},
			4,
			ancestors,
		)
		require.NoError(t, err)
		chain := populateFromPreviousStorage(scope, modifiedStorage)
		require.Equal(t, []parachaintypes.CandidateHash{modifiedCandidateAHash, candidateBHash}, chain.bestChainVec())
		require.Equal(t, 0, chain.unconnected.len())
		require.ErrorIs(t, chain.canAddCandidateAsPotential(wrongCandidateCEntry), errForkWithCandidatePendingAvailability{
			candidateHash: modifiedCandidateAHash,
		})
	})

	t.Run("multiple_pending_availability_candidates", func(t *testing.T) {
		validOptions := [][]*pendingAvailability{
			{
				{candidateHash: candidateAHash, relayParent: *relayParentAInfo},
			},
			{
				{candidateHash: candidateAHash, relayParent: *relayParentAInfo},
				{candidateHash: candidateBHash, relayParent: *relayParentBInfo},
			},
			{
				{candidateHash: candidateAHash, relayParent: *relayParentAInfo},
				{candidateHash: candidateBHash, relayParent: *relayParentBInfo},
				{candidateHash: candidateCHash, relayParent: *relayParentCInfo},
			},
		}

		for _, pending := range validOptions {
			scope, err := newScopeWithAncestors(
				*relayParentCInfo,
				baseConstraints,
				pending,
				3,
				ancestors,
			)
			require.NoError(t, err)

			chain := populateFromPreviousStorage(scope, storage)
			assert.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash}, chain.bestChainVec())
			assert.Equal(t, 0, chain.unconnected.len())
		}
	})

	t.Run("relay_parents_of_pending_availability_candidates_can_be_out_of_scope", func(t *testing.T) {
		ancestorsWithoutA := []relayChainBlockInfo{
			*relayParentBInfo,
		}

		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			[]*pendingAvailability{
				{candidateHash: candidateAHash, relayParent: *relayParentAInfo},
			},
			4,
			ancestorsWithoutA,
		)
		require.NoError(t, err)

		chain := populateFromPreviousStorage(scope, storage)
		assert.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash}, chain.bestChainVec())
		assert.Equal(t, 0, chain.unconnected.len())
	})

	t.Run("relay_parents_of_pending_availability_candidates_cannot_move_backwards", func(t *testing.T) {
		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			[]*pendingAvailability{
				{
					candidateHash: candidateAHash,
					relayParent: relayChainBlockInfo{
						Hash:        relayParentAInfo.Hash,
						Number:      1,
						StorageRoot: relayParentAInfo.StorageRoot,
					},
				},
				{
					candidateHash: candidateBHash,
					relayParent: relayChainBlockInfo{
						Hash:        relayParentBInfo.Hash,
						Number:      0,
						StorageRoot: relayParentBInfo.StorageRoot,
					},
				},
			},
			4,
			[]relayChainBlockInfo{},
		)
		require.NoError(t, err)

		chain := populateFromPreviousStorage(scope, storage)
		assert.Empty(t, chain.bestChainVec())
		assert.Equal(t, 0, chain.unconnected.len())
	})

	t.Run("more_complex_case_with_multiple_candidates_and_constraints", func(t *testing.T) {
		scope, err := newScopeWithAncestors(
			*relayParentCInfo,
			baseConstraints,
			nil,
			2,
			ancestors,
		)
		require.NoError(t, err)

		// Candidate D
		candidateDParaHead := parachaintypes.HeadData{Data: []byte{0x0e}}
		pvdD, candidateD := makeCommittedCandidate(t, paraID,
			relayParentCInfo.Hash, uint32(relayParentCInfo.Number),
			candidateCParaHead,
			candidateDParaHead,
			uint32(relayParentCInfo.Number),
		)
		candidateDHash, candidateDEntry := hashAndGetEntry(t, candidateD, pvdD, backed)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateDEntry))
		require.NoError(t, storage.addCandidateEntry(candidateDEntry))

		// Candidate F
		candidateEParaHead := parachaintypes.HeadData{Data: []byte{0x0f}}
		candidateFParaHead := parachaintypes.HeadData{Data: []byte{0xf1}}
		pvdF, candidateF := makeCommittedCandidate(t, paraID,
			relayParentCInfo.Hash, uint32(relayParentCInfo.Number),
			candidateEParaHead,
			candidateFParaHead,
			1000,
		)
		candidateFHash, candidateFEntry := hashAndGetEntry(t, candidateF, pvdF, seconded)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateFEntry))
		require.NoError(t, storage.addCandidateEntry(candidateFEntry))

		// Candidate A1
		pvdA1, candidateA1 := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			firstParachainHead,
			parachaintypes.HeadData{Data: []byte{0xb1}},
			uint32(relayParentAInfo.Number),
		)
		candidateA1Hash, candidateA1Entry := hashAndGetEntry(t, candidateA1, pvdA1, backed)

		// candidate A1 is created so that its hash is greater than the candidate A hash.
		require.Equal(t, -1, forkSelectionRule(candidateAHash, candidateA1Hash))
		require.ErrorIs(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateA1Entry),
			errForkChoiceRule{candidateHash: candidateAHash})

		require.NoError(t, storage.addCandidateEntry(candidateA1Entry))

		// Candidate B1
		pvdB1, candidateB1 := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			parachaintypes.HeadData{Data: []byte{0xb1}},
			parachaintypes.HeadData{Data: []byte{0xc1}},
			uint32(relayParentAInfo.Number),
		)
		_, candidateB1Entry := hashAndGetEntry(t, candidateB1, pvdB1, seconded)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateB1Entry))

		require.NoError(t, storage.addCandidateEntry(candidateB1Entry))

		// Candidate C1
		pvdC1, candidateC1 := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			parachaintypes.HeadData{Data: []byte{0xc1}},
			parachaintypes.HeadData{Data: []byte{0xd1}},
			uint32(relayParentAInfo.Number),
		)
		_, candidateC1Entry := hashAndGetEntry(t, candidateC1, pvdC1, backed)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateC1Entry))

		require.NoError(t, storage.addCandidateEntry(candidateC1Entry))

		// Candidate C2
		pvdC2, candidateC2 := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			parachaintypes.HeadData{Data: []byte{0xc1}},
			parachaintypes.HeadData{Data: []byte{0xd2}},
			uint32(relayParentAInfo.Number),
		)

		_, candidateC2Entry := hashAndGetEntry(t, candidateC2, pvdC2, seconded)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateC2Entry))
		require.NoError(t, storage.addCandidateEntry(candidateC2Entry))

		// Candidate A2
		candidateA2HeadData := parachaintypes.HeadData{Data: []byte{0x0c9}}
		pvdA2, candidateA2 := makeCommittedCandidate(t, paraID,
			relayParentAInfo.Hash, uint32(relayParentAInfo.Number),
			firstParachainHead,
			candidateA2HeadData,
			uint32(relayParentAInfo.Number),
		)
		candidateA2Hash, candidateA2Entry := hashAndGetEntry(t, candidateA2, pvdA2, seconded)

		require.Equal(t, -1, forkSelectionRule(candidateA2Hash, candidateAHash))
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateA2Entry))

		require.NoError(t, storage.addCandidateEntry(candidateA2Entry))

		// Candidate B2
		candidateB2HeadData := parachaintypes.HeadData{Data: []byte{0xb4}}
		pvdB2, candidateB2 := makeCommittedCandidate(t, paraID,
			relayParentBInfo.Hash, uint32(relayParentBInfo.Number),
			candidateA2HeadData,
			candidateB2HeadData,
			uint32(relayParentBInfo.Number),
		)
		candidateB2Hash, candidateB2Entry := hashAndGetEntry(t, candidateB2, pvdB2, backed)
		require.NoError(t, populateFromPreviousStorage(scope, storage).
			canAddCandidateAsPotential(candidateB2Entry))

		require.NoError(t, storage.addCandidateEntry(candidateB2Entry))

		chain := populateFromPreviousStorage(scope, storage)
		assert.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash}, chain.bestChainVec())

		unconnectedHashes := make(map[parachaintypes.CandidateHash]struct{})
		for _, unconnected := range chain.unconnected.byCandidateHash {
			unconnectedHashes[unconnected.candidateHash] = struct{}{}
		}

		expectedUnconnected := map[parachaintypes.CandidateHash]struct{}{
			candidateDHash:  {},
			candidateFHash:  {},
			candidateA2Hash: {},
			candidateB2Hash: {},
		}
		assert.Equal(t, expectedUnconnected, unconnectedHashes)

		// Cannot add as potential an already present candidate (whether it's in the best chain or in unconnected storage)
		assert.ErrorIs(t, chain.canAddCandidateAsPotential(candidateAEntry), errCandidateAlreadyKnown)
		assert.ErrorIs(t, chain.canAddCandidateAsPotential(candidateFEntry), errCandidateAlreadyKnown)

		t.Run("simulate_best_chain_reorg", func(t *testing.T) {
			// back a2, the reversion should happen at the root.
			chain := cloneFragmentChain(chain)
			chain.candidateBacked(candidateA2Hash)

			require.Equal(t, []parachaintypes.CandidateHash{candidateA2Hash, candidateB2Hash}, chain.bestChainVec())

			// candidate F is kept as it was truly unconnected. The rest will be trimmed
			unconnected := map[parachaintypes.CandidateHash]struct{}{}
			for _, entry := range chain.unconnected.byCandidateHash {
				unconnected[entry.candidateHash] = struct{}{}
			}

			require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
				candidateFHash: {},
			}, unconnected)

			// candidates A1 and A will never have potential again
			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateA1Entry), errForkChoiceRule{
				candidateHash: candidateA2Hash,
			})
			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateAEntry), errForkChoiceRule{
				candidateHash: candidateA2Hash,
			})
		})

		t.Run("simulate_more_complex_reorg", func(t *testing.T) {
			// a2 points to b2, which is backed
			// a2 has underneath a subtree a2 -> b2 -> c3 and a2 -> b2 -> c4
			// b2 and c3 are backed, c4 is kept because it has a lower candidate hash than c3
			// backing c4 will cause a chain reorg

			// candidate c3
			candidateC3HeadData := parachaintypes.HeadData{Data: []byte{0xc2}}
			candidateC3Pvd, candidateC3 := makeCommittedCandidate(t, paraID,
				relayParentBHash, uint32(relayParentBInfo.Number),
				candidateB2HeadData,
				candidateC3HeadData,
				uint32(relayParentBInfo.Number),
			)

			candidateC3Hash, candidateC3Entry := hashAndGetEntry(t, candidateC3, candidateC3Pvd, seconded)

			// candidate c4
			candidateC4HeadData := parachaintypes.HeadData{Data: []byte{0xc3}}
			candidateC4Pvd, candidateC4 := makeCommittedCandidate(t, paraID,
				relayParentBHash, uint32(relayParentBInfo.Number),
				candidateB2HeadData,
				candidateC4HeadData,
				uint32(relayParentBInfo.Number),
			)

			candidateC4Hash, candidateC4Entry := hashAndGetEntry(t, candidateC4, candidateC4Pvd, seconded)

			// c4 should have a lower candidate hash than c3
			require.Equal(t, -1, forkSelectionRule(candidateC4Hash, candidateC3Hash))

			storage := storage.clone()

			require.NoError(t, storage.addCandidateEntry(candidateC3Entry))
			require.NoError(t, storage.addCandidateEntry(candidateC4Entry))

			chain := populateFromPreviousStorage(scope, storage)

			// current best chain
			// so we will cause a reorg when backing a2 and c3
			// and trigger another reorg when backing c4
			require.Equal(t, []parachaintypes.CandidateHash{
				candidateAHash, candidateBHash, candidateCHash,
			}, chain.bestChainVec())

			chain.candidateBacked(candidateA2Hash)

			require.Equal(t, []parachaintypes.CandidateHash{
				candidateA2Hash, candidateB2Hash,
			}, chain.bestChainVec())

			chain.candidateBacked(candidateC3Hash)

			require.Equal(t, []parachaintypes.CandidateHash{
				candidateA2Hash, candidateB2Hash, candidateC3Hash,
			}, chain.bestChainVec())

			// backing c4 will cause a reorg
			chain.candidateBacked(candidateC4Hash)

			require.Equal(t, []parachaintypes.CandidateHash{
				candidateA2Hash, candidateB2Hash, candidateC4Hash,
			}, chain.bestChainVec())

			unconnected := make(map[parachaintypes.CandidateHash]struct{})
			for _, entry := range chain.unconnected.byCandidateHash {
				unconnected[entry.candidateHash] = struct{}{}
			}

			require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
				candidateFHash: {},
			}, unconnected)
		})

		// candidate F has an invalid hrmp watermark, however it was not checked beforehand
		// as we don't have its parent yet. Add its parent now (candidate E), this will not impact anything
		// as E is not yet part of the best chain.
		candidateEPvd, candidateE := makeCommittedCandidate(t, paraID,
			relayParentCHash, uint32(relayParentCInfo.Number),
			candidateDParaHead,
			candidateEParaHead,
			uint32(relayParentCInfo.Number),
		)

		candidateEHash, _ := hashAndInsertCandididate(t, storage, candidateE, candidateEPvd, seconded)
		chain = populateFromPreviousStorage(scope, storage)
		require.Equal(t, []parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash}, chain.bestChainVec())

		unconnected := make(map[parachaintypes.CandidateHash]struct{})
		for _, entry := range chain.unconnected.byCandidateHash {
			unconnected[entry.candidateHash] = struct{}{}
		}
		require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
			candidateDHash:  {},
			candidateFHash:  {},
			candidateA2Hash: {},
			candidateB2Hash: {},
			candidateEHash:  {},
		}, unconnected)

		t.Run("simulate_candidates_A_B_C_are_pending_availability", func(t *testing.T) {
			scope, err := newScopeWithAncestors(
				*relayParentCInfo, baseConstraints.Clone(),
				[]*pendingAvailability{
					{candidateHash: candidateAHash, relayParent: *relayParentAInfo},
					{candidateHash: candidateBHash, relayParent: *relayParentBInfo},
					{candidateHash: candidateCHash, relayParent: *relayParentCInfo},
				},
				2,
				ancestors,
			)
			require.NoError(t, err)

			// candidates A2, B2 will now be trimmed
			chain := populateFromPreviousStorage(scope, storage)
			require.Equal(t,
				[]parachaintypes.CandidateHash{candidateAHash, candidateBHash, candidateCHash},
				chain.bestChainVec())

			unconnectedHashes := make(map[parachaintypes.CandidateHash]struct{})
			for _, unconnected := range chain.unconnected.byCandidateHash {
				unconnectedHashes[unconnected.candidateHash] = struct{}{}
			}

			require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
				candidateDHash: {},
				candidateFHash: {},
				candidateEHash: {},
			}, unconnectedHashes)

			// cannot add as potential an already pending availability candidate
			require.ErrorIs(t, chain.canAddCandidateAsPotential(candidateAEntry), errCandidateAlreadyKnown)

			// simulate the fact that candidate A, B and C have been included
			baseConstraints := makeConstraints(0, []parachaintypes.BlockNumber{0}, parachaintypes.HeadData{Data: []byte{0x0d}})
			scope, err = newScopeWithAncestors(*relayParentCInfo, baseConstraints, nil, 2, ancestors)
			require.NoError(t, err)

			prevChain := chain
			chain = newFragmentChain(scope, newCandidateStorage())
			chain.populateFromPrevious(prevChain)
			require.Equal(t, []parachaintypes.CandidateHash{candidateDHash}, chain.bestChainVec())

			unconnectedHashes = make(map[parachaintypes.CandidateHash]struct{})
			for _, unconnected := range chain.unconnected.byCandidateHash {
				unconnectedHashes[unconnected.candidateHash] = struct{}{}
			}

			require.Equal(t, map[parachaintypes.CandidateHash]struct{}{
				candidateEHash: {},
				candidateFHash: {},
			}, unconnectedHashes)

			// mark E as backed, F will be dropped for invalid watermark.
			// empty unconnected candidates
			chain.candidateBacked(candidateEHash)
			require.Equal(t, []parachaintypes.CandidateHash{candidateDHash, candidateEHash}, chain.bestChainVec())
			require.Zero(t, chain.unconnected.len())

			var expectedErr error = &errCheckAgainstConstraints{
				fragmentValidityErr: &errOutputsInvalid{
					ModificationError: &errDisallowedHrmpWatermark{
						BlockNumber: 1000,
					},
				},
			}

			errCheckAgainstConstraints := new(errCheckAgainstConstraints)
			err = chain.canAddCandidateAsPotential(candidateFEntry)

			require.True(t, errors.As(err, errCheckAgainstConstraints))
			require.Equal(t, errCheckAgainstConstraints, expectedErr)
		})
	})
}

func cloneFragmentChain(original *fragmentChain) *fragmentChain {
	// Clone the scope
	clonedScope := &scope{
		relayParent:         original.scope.relayParent,
		baseConstraints:     original.scope.baseConstraints.Clone(),
		pendingAvailability: append([]*pendingAvailability(nil), original.scope.pendingAvailability...),
		maxDepth:            original.scope.maxDepth,
		ancestors:           original.scope.ancestors.Copy(),
		ancestorsByHash:     make(map[common.Hash]relayChainBlockInfo),
	}

	for k, v := range original.scope.ancestorsByHash {
		clonedScope.ancestorsByHash[k] = v
	}

	// Clone the best chain
	clonedBestChain := newBackedChain()
	for _, node := range original.bestChain.chain {
		clonedNode := &fragmentNode{
			fragment:                node.fragment,
			candidateHash:           node.candidateHash,
			parentHeadDataHash:      node.parentHeadDataHash,
			outputHeadDataHash:      node.outputHeadDataHash,
			cumulativeModifications: node.cumulativeModifications.Clone(),
		}
		clonedBestChain.push(clonedNode)
	}

	// Clone the unconnected storage
	clonedUnconnected := original.unconnected.clone()

	// Create the cloned fragment chain
	clonedFragmentChain := &fragmentChain{
		scope:       clonedScope,
		bestChain:   clonedBestChain,
		unconnected: clonedUnconnected,
	}

	return clonedFragmentChain
}

func TestFindAncestorPathAndFindBackableChainEmptyBestChain(t *testing.T) {
	relayParent := common.BytesToHash(bytes.Repeat([]byte{1}, 32))
	requiredParent := parachaintypes.HeadData{Data: []byte{0xff}}
	maxDepth := uint(10)

	// Empty chain
	baseConstraints := makeConstraints(0, []parachaintypes.BlockNumber{0}, requiredParent)

	relayParentInfo := relayChainBlockInfo{
		Number:      0,
		Hash:        relayParent,
		StorageRoot: common.Hash{},
	}

	scope, err := newScopeWithAncestors(relayParentInfo, baseConstraints, nil, maxDepth, nil)
	require.NoError(t, err)

	chain := newFragmentChain(scope, newCandidateStorage())
	assert.Equal(t, 0, chain.bestChainLen())

	assert.Equal(t, 0, chain.findAncestorPath(map[parachaintypes.CandidateHash]struct{}{}))
	assert.Equal(t, []*candidateAndRelayParent{}, chain.findBackableChain(map[parachaintypes.CandidateHash]struct{}{}, 2))

	// Invalid candidate
	ancestors := map[parachaintypes.CandidateHash]struct{}{
		{Value: common.Hash{}}: {},
	}
	assert.Equal(t, 0, chain.findAncestorPath(ancestors))
	assert.Equal(t, []*candidateAndRelayParent{}, chain.findBackableChain(ancestors, 2))
}

func TestFindAncestorPathAndFindBackableChain(t *testing.T) {
	paraID := parachaintypes.ParaID(5)
	relayParent := common.BytesToHash(bytes.Repeat([]byte{1}, 32))
	requiredParent := parachaintypes.HeadData{Data: []byte{0xff}}
	maxDepth := uint(5)
	relayParentNumber := uint32(0)
	relayParentStorageRoot := common.Hash{}

	type CandidateAndPVD struct {
		candidate parachaintypes.CommittedCandidateReceipt
		pvd       parachaintypes.PersistedValidationData
	}

	candidates := make([]*CandidateAndPVD, 0)

	// candidate 0
	candidate0Pvd, candidate0 := makeCommittedCandidate(t, paraID,
		relayParent, 0, requiredParent, parachaintypes.HeadData{Data: []byte{0x00}}, relayParentNumber)
	candidates = append(candidates, &CandidateAndPVD{candidate: candidate0, pvd: candidate0Pvd})

	// candidate 1 to 5
	for i := 1; i <= 5; i++ {
		candidatePvd, candidate := makeCommittedCandidate(t, paraID,
			relayParent, 0,
			parachaintypes.HeadData{Data: []byte{byte(i - 1)}},
			parachaintypes.HeadData{Data: []byte{byte(i)}},
			relayParentNumber)
		candidates = append(candidates, &CandidateAndPVD{candidate: candidate, pvd: candidatePvd})
	}

	storage := newCandidateStorage()

	for _, c := range candidates {
		candidateHash, err := c.candidate.Hash()
		require.NoError(t, err)

		entry, err := newCandidateEntry(parachaintypes.CandidateHash{Value: candidateHash}, c.candidate, c.pvd, seconded)
		require.NoError(t, err)

		err = storage.addCandidateEntry(entry)
		require.NoError(t, err)
	}

	candidateHashes := make([]parachaintypes.CandidateHash, 0)
	for _, c := range candidates {
		candidateHash, err := c.candidate.Hash()
		require.NoError(t, err)
		candidateHashes = append(candidateHashes, parachaintypes.CandidateHash{Value: candidateHash})
	}

	type Ancestors = map[parachaintypes.CandidateHash]struct{}

	hashes := func(from, to uint) []*candidateAndRelayParent {
		var output []*candidateAndRelayParent

		for i := from; i < to; i++ {
			output = append(output, &candidateAndRelayParent{
				candidateHash:   candidateHashes[i],
				realyParentHash: relayParent,
			})
		}

		return output
	}

	relayParentInfo := relayChainBlockInfo{
		Number:      parachaintypes.BlockNumber(relayParentNumber),
		Hash:        relayParent,
		StorageRoot: relayParentStorageRoot,
	}

	baseConstraints := makeConstraints(0, []parachaintypes.BlockNumber{0}, requiredParent)
	scope, err := newScopeWithAncestors(
		relayParentInfo,
		baseConstraints,
		nil,
		maxDepth,
		nil,
	)
	require.NoError(t, err)

	chain := populateFromPreviousStorage(scope, storage)

	// for now candidates are only seconded, not backed, the best chain is empty
	// and no candidate will be returned

	require.Equal(t, 6, len(candidateHashes))
	require.Equal(t, 0, chain.bestChainLen())
	require.Equal(t, 6, chain.unconnected.len())

	for count := 0; count < 10; count++ {
		require.Equal(t, 0, len(chain.findBackableChain(make(Ancestors), uint32(count))))
	}

	t.Run("couple_candidates_backed", func(t *testing.T) {
		chain := cloneFragmentChain(chain)
		chain.candidateBacked(candidateHashes[5])

		for count := 0; count < 10; count++ {
			require.Equal(t, 0, len(chain.findBackableChain(make(Ancestors), uint32(count))))
		}

		chain.candidateBacked(candidateHashes[3])
		chain.candidateBacked(candidateHashes[4])

		for count := 0; count < 10; count++ {
			require.Equal(t, 0, len(chain.findBackableChain(make(Ancestors), uint32(count))))
		}

		chain.candidateBacked(candidateHashes[1])

		for count := 0; count < 10; count++ {
			require.Equal(t, 0, len(chain.findBackableChain(make(Ancestors), uint32(count))))
		}

		chain.candidateBacked(candidateHashes[0])
		require.Equal(t, hashes(0, 1), chain.findBackableChain(make(Ancestors), 1))

		for c := 2; c < 10; c++ {
			require.Equal(t, hashes(0, 2), chain.findBackableChain(make(Ancestors), uint32(c)))
		}

		// now back the missing piece
		chain.candidateBacked(candidateHashes[2])
		require.Equal(t, 6, chain.bestChainLen())

		for count := 0; count < 10; count++ {
			var result []*candidateAndRelayParent
			if count > 6 {
				result = hashes(0, 6)
			} else {
				for i := 0; i < count && i < 6; i++ {
					result = append(result, &candidateAndRelayParent{
						candidateHash:   candidateHashes[i],
						realyParentHash: relayParent,
					})
				}
			}
			require.Equal(t, result, chain.findBackableChain(make(Ancestors), uint32(count)))
		}
	})

	t.Run("back_all_candidates_in_random_order", func(t *testing.T) {
		candidatesShuffled := make([]parachaintypes.CandidateHash, len(candidateHashes))
		for i := range candidateHashes {
			candidatesShuffled[i] = parachaintypes.CandidateHash{
				Value: common.NewHash(candidateHashes[i].Value.ToBytes()),
			}
		}

		rand.Shuffle(len(candidatesShuffled), func(i, j int) {
			candidatesShuffled[i], candidatesShuffled[j] = candidatesShuffled[j], candidatesShuffled[i]
		})

		for _, c := range candidatesShuffled {
			chain.candidateBacked(c)
			storage.markBacked(c)
		}

		// no ancestors supplied
		require.Equal(t, 0, chain.findAncestorPath(make(Ancestors)))
		require.Equal(t, []*candidateAndRelayParent(nil), chain.findBackableChain(make(Ancestors), 0))
		require.Equal(t, hashes(0, 1), chain.findBackableChain(make(Ancestors), 1))
		require.Equal(t, hashes(0, 2), chain.findBackableChain(make(Ancestors), 2))
		require.Equal(t, hashes(0, 5), chain.findBackableChain(make(Ancestors), 5))

		for count := 6; count < 10; count++ {
			backableChain := chain.findBackableChain(make(Ancestors), uint32(count))
			require.Equal(t, hashes(0, 6), backableChain)
		}

		// ancestors which is not part of the chain will be ignored
		ancestors := make(Ancestors)
		ancestors[parachaintypes.CandidateHash{Value: common.Hash{}}] = struct{}{}
		require.Equal(t, 0, chain.findAncestorPath(ancestors))
		require.Equal(t, hashes(0, 4), chain.findBackableChain(ancestors, 4))

		ancestors = make(Ancestors)
		ancestors[candidateHashes[1]] = struct{}{}
		ancestors[parachaintypes.CandidateHash{Value: common.Hash{}}] = struct{}{}
		require.Equal(t, 0, chain.findAncestorPath(ancestors))
		require.Equal(t, hashes(0, 4), chain.findBackableChain(ancestors, 4))

		ancestors = make(Ancestors)
		ancestors[candidateHashes[0]] = struct{}{}
		ancestors[parachaintypes.CandidateHash{Value: common.Hash{}}] = struct{}{}
		require.Equal(t, 1, chain.findAncestorPath(maps.Clone(ancestors)))
		require.Equal(t, hashes(1, 5), chain.findBackableChain(ancestors, 4))

		// ancestors which are part of the chain but don't form a path from root, will be ignored
		ancestors = make(Ancestors)
		ancestors[candidateHashes[1]] = struct{}{}
		ancestors[candidateHashes[2]] = struct{}{}
		require.Equal(t, 0, chain.findAncestorPath(maps.Clone(ancestors)))
		require.Equal(t, hashes(0, 4), chain.findBackableChain(ancestors, 4))

		// valid ancestors
		ancestors = make(Ancestors)
		ancestors[candidateHashes[2]] = struct{}{}
		ancestors[candidateHashes[0]] = struct{}{}
		ancestors[candidateHashes[1]] = struct{}{}
		require.Equal(t, 3, chain.findAncestorPath(maps.Clone(ancestors)))
		require.Equal(t, hashes(3, 5), chain.findBackableChain(maps.Clone(ancestors), 2))

		for count := 3; count < 10; count++ {
			require.Equal(t, hashes(3, 6), chain.findBackableChain(maps.Clone(ancestors), uint32(count)))
		}

		// valid ancestors with candidates which have been omitted due to timeouts
		ancestors = make(Ancestors)
		ancestors[candidateHashes[0]] = struct{}{}
		ancestors[candidateHashes[2]] = struct{}{}
		require.Equal(t, 1, chain.findAncestorPath(maps.Clone(ancestors)))
		require.Equal(t, hashes(1, 4), chain.findBackableChain(maps.Clone(ancestors), 3))
		require.Equal(t, hashes(1, 5), chain.findBackableChain(maps.Clone(ancestors), 4))

		for count := 5; count < 10; count++ {
			require.Equal(t, hashes(1, 6), chain.findBackableChain(maps.Clone(ancestors), uint32(count)))
		}

		ancestors = make(Ancestors)
		ancestors[candidateHashes[0]] = struct{}{}
		ancestors[candidateHashes[1]] = struct{}{}
		ancestors[candidateHashes[3]] = struct{}{}
		require.Equal(t, 2, chain.findAncestorPath(maps.Clone(ancestors)))
		require.Equal(t, hashes(2, 6), chain.findBackableChain(maps.Clone(ancestors), 4))

		require.Equal(t, hashes(0, 0), chain.findBackableChain(maps.Clone(ancestors), 0))

		// stop when we've found a candidate which is pending availability
		scope, err := newScopeWithAncestors(relayParentInfo, baseConstraints,
			[]*pendingAvailability{
				{candidateHash: candidateHashes[3], relayParent: relayParentInfo},
			},
			maxDepth,
			nil,
		)
		require.NoError(t, err)
		chain = populateFromPreviousStorage(scope, storage)
		ancestors = make(Ancestors)
		ancestors[candidateHashes[0]] = struct{}{}
		ancestors[candidateHashes[1]] = struct{}{}
		require.Equal(t, hashes(2, 3), chain.findBackableChain(maps.Clone(ancestors), 3))
	})
}
