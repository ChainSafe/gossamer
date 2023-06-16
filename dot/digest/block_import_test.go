// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestBlockImportHandle(t *testing.T) {
	mockedError := errors.New("mock error")
	cases := map[string]struct {
		createBlockHeader func(*testing.T) (*types.Header, []types.ConsensusDigest)
		setupGrandpaState func(*testing.T, *gomock.Controller, *types.Header, []types.ConsensusDigest) GrandpaState
		setupEpochState   func(*testing.T, *gomock.Controller, *types.Header, []types.ConsensusDigest) EpochState
		wantErr           error
		errString         string
	}{
		"handle_babe_digest_fails": {
			wantErr:   mockedError,
			errString: "while handling digests: while handling babe digest: mock error",
			setupGrandpaState: func(*testing.T, *gomock.Controller, *types.Header,
				[]types.ConsensusDigest) GrandpaState {
				return nil
			},
			setupEpochState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) EpochState {
				epochStateMock := NewMockEpochState(ctrl)

				expectedBabeConsensusDigest := types.NewBabeConsensusDigest()
				err := scale.Unmarshal(digestData[0].Data, &expectedBabeConsensusDigest)
				require.NoError(t, err)

				epochStateMock.EXPECT().
					HandleBABEDigest(header, expectedBabeConsensusDigest).
					Return(mockedError)
				return epochStateMock
			},
			createBlockHeader: func(t *testing.T) (*types.Header, []types.ConsensusDigest) {
				_, _, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
				keyring, _ := keystore.NewSr25519Keyring()

				keyPairs := []*sr25519.Keypair{
					keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
				}

				authorities := make([]types.AuthorityRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					authorities[i] = types.AuthorityRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createNextEpoch := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextEpochData{
						Authorities: authorities[:],
						Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
					})
				}

				createNextConfigData := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextConfigDataV1{
						C1:             9,
						C2:             10,
						SecondarySlots: 1,
					})
				}

				return createBlockWithDigests(t, &genesisHeader,
					createNextEpoch, createNextConfigData)
			},
		},
		"handle_grandpa_digest_fails": {
			wantErr:   mockedError,
			errString: "while handling digests: while handling grandpa digest: mock error",
			setupGrandpaState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) GrandpaState {

				expectedGrandpaConsensusDigest := types.NewGrandpaConsensusDigest()
				err := scale.Unmarshal(digestData[0].Data, &expectedGrandpaConsensusDigest)
				require.NoError(t, err)

				grandpaStateMock := NewMockGrandpaState(ctrl)
				grandpaStateMock.EXPECT().
					HandleGRANDPADigest(header, expectedGrandpaConsensusDigest).
					Return(mockedError)
				return grandpaStateMock
			},
			setupEpochState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) EpochState {
				epochStateMock := NewMockEpochState(ctrl)

				for _, consensusDigest := range digestData {
					expectedBabeConsensusDigest := types.NewBabeConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedBabeConsensusDigest)
					require.NoError(t, err)

					epochStateMock.EXPECT().
						HandleBABEDigest(header, expectedBabeConsensusDigest).
						Return(nil)
				}

				return epochStateMock
			},
			createBlockHeader: func(t *testing.T) (*types.Header, []types.ConsensusDigest) {
				_, _, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
				keyring, _ := keystore.NewSr25519Keyring()

				keyPairs := []*sr25519.Keypair{
					keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
				}

				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				authorities := make([]types.AuthorityRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					authorities[i] = types.AuthorityRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}

					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createNextEpoch := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextEpochData{
						Authorities: authorities[:],
						Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
					})
				}

				createNextConfigData := func() types.ConsensusDigest {
					versionedNextConfigData := types.NewVersionedNextConfigData()
					versionedNextConfigData.Set(types.NextConfigDataV1{
						C1:             9,
						C2:             10,
						SecondarySlots: 1,
					})

					return createBABEConsensusDigest(t, versionedNextConfigData)
				}

				createScheduledChange := func() types.ConsensusDigest {
					return createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
						Auths: grandpaAuths[:1],
						Delay: 2,
					})
				}

				return createBlockWithDigests(t, &genesisHeader,
					createNextEpoch, createNextConfigData, createScheduledChange)
			},
		},
		"handle_babe_and_grandpa_digests_successfully": {
			setupGrandpaState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) GrandpaState {

				grandpaStateMock := NewMockGrandpaState(ctrl)
				for _, consensusDigest := range digestData {
					expectedGrandpaConsensusDigest := types.NewGrandpaConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedGrandpaConsensusDigest)
					require.NoError(t, err)

					grandpaStateMock.EXPECT().
						HandleGRANDPADigest(header, expectedGrandpaConsensusDigest).
						Return(nil)
				}

				grandpaStateMock.EXPECT().ApplyForcedChanges(header).Return(nil)
				return grandpaStateMock
			},
			setupEpochState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) EpochState {
				epochStateMock := NewMockEpochState(ctrl)

				for _, consensusDigest := range digestData {
					expectedBabeConsensusDigest := types.NewBabeConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedBabeConsensusDigest)
					require.NoError(t, err)

					epochStateMock.EXPECT().
						HandleBABEDigest(header, expectedBabeConsensusDigest).
						Return(nil)
				}

				return epochStateMock
			},
			createBlockHeader: func(t *testing.T) (*types.Header, []types.ConsensusDigest) {
				_, _, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
				keyring, _ := keystore.NewSr25519Keyring()

				keyPairs := []*sr25519.Keypair{
					keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
				}

				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				authorities := make([]types.AuthorityRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					authorities[i] = types.AuthorityRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}

					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createNextEpoch := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextEpochData{
						Authorities: authorities[:],
						Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
					})
				}

				createNextConfigData := func() types.ConsensusDigest {
					versionedNextConfigData := types.NewVersionedNextConfigData()
					versionedNextConfigData.Set(types.NextConfigDataV1{
						C1:             9,
						C2:             10,
						SecondarySlots: 1,
					})

					return createBABEConsensusDigest(t, versionedNextConfigData)
				}

				createScheduledChange := func() types.ConsensusDigest {
					return createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
						Auths: grandpaAuths[:1],
						Delay: 2,
					})
				}

				return createBlockWithDigests(t, &genesisHeader,
					createNextEpoch, createNextConfigData, createScheduledChange)
			},
		},
		"handle_unknown_consensus_id_should_be_succesfull": {
			setupGrandpaState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) GrandpaState {

				grandpaStateMock := NewMockGrandpaState(ctrl)
				for _, consensusDigest := range digestData {
					expectedGrandpaConsensusDigest := types.NewGrandpaConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedGrandpaConsensusDigest)
					require.NoError(t, err)

					grandpaStateMock.EXPECT().
						HandleGRANDPADigest(header, expectedGrandpaConsensusDigest).
						Return(nil)
				}

				grandpaStateMock.EXPECT().ApplyForcedChanges(header).Return(nil)
				return grandpaStateMock
			},
			setupEpochState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) EpochState {
				epochStateMock := NewMockEpochState(ctrl)

				// we expect to handle only one consensus digest since the
				// other one contains a different consensus engine id
				expectedBabeConsensusDigest := types.NewBabeConsensusDigest()
				err := scale.Unmarshal(digestData[0].Data, &expectedBabeConsensusDigest)
				require.NoError(t, err)

				epochStateMock.EXPECT().
					HandleBABEDigest(header, expectedBabeConsensusDigest).
					Return(nil)

				return epochStateMock
			},
			createBlockHeader: func(t *testing.T) (*types.Header, []types.ConsensusDigest) {
				_, _, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
				keyring, _ := keystore.NewSr25519Keyring()

				keyPairs := []*sr25519.Keypair{
					keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
				}

				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				authorities := make([]types.AuthorityRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					authorities[i] = types.AuthorityRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}

					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createNextEpoch := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextEpochData{
						Authorities: authorities[:],
						Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
					})
				}

				// change the nextConfigData consensus engine id
				createNextConfigData := func() types.ConsensusDigest {
					versionedNextConfigData := types.NewVersionedNextConfigData()
					versionedNextConfigData.Set(types.NextConfigDataV1{
						C1:             9,
						C2:             10,
						SecondarySlots: 1,
					})

					consensusDigest := createBABEConsensusDigest(t, versionedNextConfigData)
					consensusDigest.ConsensusEngineID = [4]byte{0, 0, 0, 0}
					return consensusDigest
				}

				createScheduledChange := func() types.ConsensusDigest {
					return createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
						Auths: grandpaAuths[:1],
						Delay: 2,
					})
				}

				return createBlockWithDigests(t, &genesisHeader,
					createNextEpoch, createNextConfigData, createScheduledChange)
			},
		},
		"on_block_import_failed_to_apply_forced_changes": {
			wantErr:   mockedError,
			errString: "while applying forced changes: mock error",
			setupGrandpaState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) GrandpaState {

				grandpaStateMock := NewMockGrandpaState(ctrl)
				for _, consensusDigest := range digestData {
					expectedGrandpaConsensusDigest := types.NewGrandpaConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedGrandpaConsensusDigest)
					require.NoError(t, err)

					grandpaStateMock.EXPECT().
						HandleGRANDPADigest(header, expectedGrandpaConsensusDigest).
						Return(nil)
				}

				grandpaStateMock.EXPECT().ApplyForcedChanges(header).Return(mockedError)
				return grandpaStateMock
			},
			setupEpochState: func(t *testing.T, ctrl *gomock.Controller, header *types.Header,
				digestData []types.ConsensusDigest) EpochState {
				epochStateMock := NewMockEpochState(ctrl)

				for _, consensusDigest := range digestData {
					expectedBabeConsensusDigest := types.NewBabeConsensusDigest()
					err := scale.Unmarshal(consensusDigest.Data, &expectedBabeConsensusDigest)
					require.NoError(t, err)

					epochStateMock.EXPECT().
						HandleBABEDigest(header, expectedBabeConsensusDigest).
						Return(nil)
				}

				return epochStateMock
			},
			createBlockHeader: func(t *testing.T) (*types.Header, []types.ConsensusDigest) {
				_, _, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
				keyring, _ := keystore.NewSr25519Keyring()

				keyPairs := []*sr25519.Keypair{
					keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
				}

				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				authorities := make([]types.AuthorityRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					authorities[i] = types.AuthorityRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}

					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createNextEpoch := func() types.ConsensusDigest {
					return createBABEConsensusDigest(t, types.NextEpochData{
						Authorities: authorities[:],
						Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
					})
				}

				createNextConfigData := func() types.ConsensusDigest {
					versionedNextConfigData := types.NewVersionedNextConfigData()
					versionedNextConfigData.Set(types.NextConfigDataV1{
						C1:             9,
						C2:             10,
						SecondarySlots: 1,
					})

					return createBABEConsensusDigest(t, versionedNextConfigData)
				}

				createScheduledChange := func() types.ConsensusDigest {
					return createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
						Auths: grandpaAuths[:1],
						Delay: 2,
					})
				}

				return createBlockWithDigests(t, &genesisHeader,
					createNextEpoch, createNextConfigData, createScheduledChange)
			},
		},
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			importedHeader, consensusDigests := tt.createBlockHeader(t)

			ctrl := gomock.NewController(t)
			// idexes 0 and 1 belongs to the BABE digests next epoch data and next config data respectively
			// the indexes after that are for GRANDPA scheduled change and forced change
			epochStateMock := tt.setupEpochState(t, ctrl, importedHeader, consensusDigests[:2])
			grandpaStateMock := tt.setupGrandpaState(t, ctrl, importedHeader, consensusDigests[2:])

			onBlockImportDigestHandler := NewBlockImportHandler(epochStateMock, grandpaStateMock)
			err := onBlockImportDigestHandler.Handle(importedHeader)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.errString != "" {
				require.EqualError(t, err, tt.errString)
			}
		})
	}
}

type withDigest func() types.ConsensusDigest

func createBlockWithDigests(t *testing.T, genesisHeader *types.Header, digestsToApply ...withDigest) (
	header *types.Header, consensusDigests []types.ConsensusDigest) {
	t.Helper()

	digest := types.NewDigest()
	consensusDigests = make([]types.ConsensusDigest, len(digestsToApply))
	digestAddArgs := make([]scale.VaryingDataTypeValue, len(digestsToApply))

	for idx, createDigestFn := range digestsToApply {
		consensusDigest := createDigestFn()
		consensusDigests[idx] = consensusDigest
		digestAddArgs[idx] = consensusDigest
	}

	err := digest.Add(digestAddArgs...)
	require.NoError(t, err)

	return &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     1,
		Digest:     digest,
	}, consensusDigests
}

func createBABEConsensusDigest(t *testing.T, digestData scale.VaryingDataTypeValue) types.ConsensusDigest {
	t.Helper()

	babeConsensusDigest := types.NewBabeConsensusDigest()
	require.NoError(t, babeConsensusDigest.Set(digestData))

	marshaledData, err := scale.Marshal(babeConsensusDigest)
	require.NoError(t, err)

	return types.ConsensusDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              marshaledData,
	}
}

func createGRANDPAConsensusDigest(t *testing.T, digestData scale.VaryingDataTypeValue) types.ConsensusDigest {
	t.Helper()

	grandpaConsensusDigest := types.NewGrandpaConsensusDigest()
	require.NoError(t, grandpaConsensusDigest.Set(digestData))

	marshaledData, err := scale.Marshal(grandpaConsensusDigest)
	require.NoError(t, err)

	return types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              marshaledData,
	}
}
