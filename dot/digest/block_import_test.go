// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func decodePreRuntimeAndBabeConsensus(t *testing.T, preRuntimeBytes, babeConsensusBytes []byte) (
	preRuntime scale.VaryingDataTypeValue, babeConsensusDigest scale.VaryingDataType) {

	babeDigest := types.NewBabeDigest()
	err := scale.Unmarshal(preRuntimeBytes, &babeDigest)
	require.NoError(t, err)

	preRuntimeBabeDigest, err := babeDigest.Value()
	require.NoError(t, err)

	babeConsensusDigest = types.NewBabeConsensusDigest()
	scale.Unmarshal(babeConsensusBytes, &babeConsensusDigest)

	return preRuntimeBabeDigest, babeConsensusDigest
}

func TestDecodeConsensusMessage(t *testing.T) {
	const firstSlot = uint64(264379767)

	babePreRuntimeBlock10173960 := common.MustHexToBytes("0x0308000000976d60100000000018d2c35ec8e484d27162571ae544a5f0ae3e1c35d4e5ae31cddc00004c59b10cb2ad8983100a2af537582021416498d6d4fa32f04703d83cd4f809b218ee3b0be6bb118a5905ececb0b6af15a2f77280870848ea2a97f23e930caaf64e18c30c")
	babeConsensusDigestBlock10173960 := common.MustHexToBytes("0x0140a2f559499573b5878287f8cc18abfb5d070229a6" +
		"584a125113725a38ba62996f0100000000000" +
		"000886a9fa1a9ee1cc5666fa0b6d5c16754617b923c36442" +
		"8ad51c79e6b52ad1d3e0100000000000000524c2bc612c5a" +
		"6f22845e6a80944db9e197beaa50c242f34a2a45a3666a2eb7" +
		"a01000000000000002cc19bd8a23c534f9f2ae91651c84bc2fd" +
		"ef757e8aca156edc5e92eeb589043d010000000000000036484c" +
		"5ae34f2fcd3a56a9ac7681a183ec1628e495dd142d530d10e4e8c" +
		"1432401000000000000006e4ada50950a8e35392dcf101c0c82b29" +
		"e95e3da2d71d7e0909fde502de0ca5b0100000000000000f08a1a9" +
		"eb8536a735b23b04d3d6501bc720d0d6a40a89c911a771f1c2d7a4260" +
		"0100000000000000eeb16c9f7722516fca6b3c1bc0fab270a5a3280f5" +
		"1ce1b21e834d8a7f3d452210100000000000000b24ced05d1e1fbadf98" +
		"791e452f9d265a3118e9828a9ce0f64ed73d91764ea1701000000000000" +
		"0088c33bf243528e684e9b41c99abc2083e0d6cd391bcb2d8340723b12b" +
		"4706514010000000000000056646045eeaee5ac194bd2436420c9070d81ba" +
		"0d17624832c1eb12c3b353c6480100000000000000565bea8cf9db123c71a5" +
		"015380a98560839b129030287e2008dc8337affc4b790100000000000000ac5" +
		"c5c355619970c9c2049ace0551338e6f345ad01547338a90f317cc043a820010" +
		"0000000000000642b60bdaaa80105756645e9d1e267ac7c384a73ed004f65e91" +
		"180f99b49ef06010000000000000034e9be26ef44e368d4d438e2782b5b0805fc4" +
		"0257c7436026af4113a80245e530100000000000000c48aee5296e2911c4b3a9051" +
		"fe0a4c4ae81741e110b6ae3e89b138cd874d90290100000000000000d4922aff3b" +
		"5533f233e84356050140d7461f2ba6415034ca2d7a95154437cf3e")

	preRuntimeBlock10173360 := common.MustHexToBytes("0x030e0000003f6b6010000000008ca3793fe52f04edef0c90f6aa1569db7c0a1f99d151c93bc0287e9056a31466735bbfc56a7b72c51455bf61fb308493de6f72c2a5002723175bbd39c221f90b76915ab4e894a990fa5495f95305c69b87a5dc054500a77b7eaf90ed9ebf180d")
	babeConsensusDigestBlock10173360 := common.MustHexToBytes("0x0140a2f559499573b5878287f8cc18abfb5d070229a6584a125113725a38ba62996f0100000000000000886a9fa1a9ee1cc5666fa0b6d5c16754617b923c364428ad51c79e6b52ad1d3e0100000000000000524c2bc612c5a6f22845e6a80944db9e197beaa50c242f34a2a45a3666a2eb7a01000000000000002cc19bd8a23c534f9f2ae91651c84bc2fdef757e8aca156edc5e92eeb589043d010000000000000036484c5ae34f2fcd3a56a9ac7681a183ec1628e495dd142d530d10e4e8c1432401000000000000006e4ada50950a8e35392dcf101c0c82b29e95e3da2d71d7e0909fde502de0ca5b0100000000000000f08a1a9eb8536a735b23b04d3d6501bc720d0d6a40a89c911a771f1c2d7a42600100000000000000eeb16c9f7722516fca6b3c1bc0fab270a5a3280f51ce1b21e834d8a7f3d452210100000000000000b24ced05d1e1fbadf98791e452f9d265a3118e9828a9ce0f64ed73d91764ea17010000000000000088c33bf243528e684e9b41c99abc2083e0d6cd391bcb2d8340723b12b4706514010000000000000056646045eeaee5ac194bd2436420c9070d81ba0d17624832c1eb12c3b353c6480100000000000000565bea8cf9db123c71a5015380a98560839b129030287e2008dc8337affc4b790100000000000000ac5c5c355619970c9c2049ace0551338e6f345ad01547338a90f317cc043a8200100000000000000642b60bdaaa80105756645e9d1e267ac7c384a73ed004f65e91180f99b49ef06010000000000000034e9be26ef44e368d4d438e2782b5b0805fc40257c7436026af4113a80245e530100000000000000c48aee5296e2911c4b3a9051fe0a4c4ae81741e110b6ae3e89b138cd874d90290100000000000000318f653fc8b38720d0599abd6c3ff2ab63e8c57db0ec0117db69f9d7cbaab378")

	fmt.Printf("==== #10173960 ======\n")
	preRuntimeBabeDigest, babeConsensusDigest := decodePreRuntimeAndBabeConsensus(t, babePreRuntimeBlock10173960, babeConsensusDigestBlock10173960)
	preRuntime := preRuntimeBabeDigest.(types.BabeSecondaryVRFPreDigest)

	// 274754967 - 264379767 = 10375200
	// 10375200 / 600 = 17292
	fmt.Println(preRuntime.SlotNumber)
	fmt.Println((preRuntime.SlotNumber - firstSlot) / 600)

	scale.Unmarshal(babeConsensusDigestBlock10173960, &babeConsensusDigest)
	fmt.Println(babeConsensusDigest.String())

	fmt.Printf("==== #10173360 ======\n")
	preRuntimeBabeDigest, babeConsensusDigest = decodePreRuntimeAndBabeConsensus(t, preRuntimeBlock10173360, babeConsensusDigestBlock10173360)
	preRuntime, ok := preRuntimeBabeDigest.(types.BabeSecondaryVRFPreDigest)
	require.Truef(t, ok, "right type is %T", preRuntimeBabeDigest)

	// 274754967 - 264379767 = 10375200
	// 10375200 / 600 = 17292
	fmt.Println(preRuntime.SlotNumber)
	fmt.Println((preRuntime.SlotNumber - firstSlot) / 600)

	scale.Unmarshal(babeConsensusDigestBlock10173960, &babeConsensusDigest)
	fmt.Println(babeConsensusDigest.String())
}

func TestBlockImportHandle(t *testing.T) {
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

	genericNextEpochDigest := createBABEConsensusDigest(t, types.NextEpochData{
		Authorities: authorities,
		Randomness:  [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8},
	})

	versionedNextConfigData := types.NewVersionedNextConfigData()
	versionedNextConfigData.Set(types.NextConfigDataV1{
		C1:             9,
		C2:             10,
		SecondarySlots: 1,
	})
	genericNextConfigDataDigest := createBABEConsensusDigest(t, versionedNextConfigData)

	mockedError := errors.New("mock error")
	cases := map[string]struct {
		createBlockHeader func(*testing.T) (*types.Header, []types.ConsensusDigest)
		setupGrandpaState func(*testing.T, *gomock.Controller, *types.Header, []types.ConsensusDigest) GrandpaState
		setupEpochState   func(*testing.T, *gomock.Controller, *types.Header, []types.ConsensusDigest) EpochState
		wantErr           error
		errString         string
	}{
		"handle_babe_digest_fails": {
			wantErr: mockedError,
			errString: "consensus digests: " +
				"handling babe digest: mock error",
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

				consensusDigests := []types.ConsensusDigest{
					genericNextEpochDigest, genericNextConfigDataDigest,
				}
				return createBlockWithDigests(t, &genesisHeader,
						genericNextEpochDigest, genericNextConfigDataDigest),
					consensusDigests
			},
		},
		"handle_grandpa_digest_fails": {
			wantErr: mockedError,
			errString: "consensus digests: " +
				"handling grandpa digest: mock error",
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
				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createScheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
					Auths: grandpaAuths[:1],
					Delay: 2,
				})

				consensusDigests := []types.ConsensusDigest{
					genericNextEpochDigest,
					genericNextConfigDataDigest,
					createScheduledChange,
				}
				return createBlockWithDigests(t, &genesisHeader,
						genericNextEpochDigest,
						genericNextConfigDataDigest,
						createScheduledChange),
					consensusDigests
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
				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				createScheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
					Auths: grandpaAuths[:1],
					Delay: 2,
				})

				consensusDigests := []types.ConsensusDigest{
					genericNextEpochDigest, genericNextConfigDataDigest, createScheduledChange,
				}
				return createBlockWithDigests(t, &genesisHeader,
						genericNextEpochDigest, genericNextConfigDataDigest, createScheduledChange),
					consensusDigests
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
				grandpaAuths := make([]types.GrandpaAuthoritiesRaw, len(keyPairs))
				for i, keyPair := range keyPairs {
					grandpaAuths[i] = types.GrandpaAuthoritiesRaw{
						Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
					}
				}

				versionedNextConfigData := types.NewVersionedNextConfigData()
				versionedNextConfigData.Set(types.NextConfigDataV1{
					C1:             9,
					C2:             10,
					SecondarySlots: 1,
				})

				wrongEngineNextConfigData := createBABEConsensusDigest(t, versionedNextConfigData)
				// change the nextConfigData consensus engine id
				wrongEngineNextConfigData.ConsensusEngineID = [4]byte{0, 0, 0, 0}

				createScheduledChange := createGRANDPAConsensusDigest(t, types.GrandpaScheduledChange{
					Auths: grandpaAuths[:1],
					Delay: 2,
				})

				consensusDigests := []types.ConsensusDigest{
					genericNextConfigDataDigest, wrongEngineNextConfigData, createScheduledChange,
				}
				return createBlockWithDigests(t, &genesisHeader,
						genericNextConfigDataDigest,
						wrongEngineNextConfigData,
						createScheduledChange),
					consensusDigests
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
			err := onBlockImportDigestHandler.HandleDigests(importedHeader)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.errString != "" {
				require.EqualError(t, err, tt.errString)
			}
		})
	}
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

func createBlockWithDigests(t *testing.T, genesisHeader *types.Header, digestsToApply ...types.ConsensusDigest) (
	header *types.Header) {
	t.Helper()

	digest := types.NewDigest()
	digestAddArgs := make([]scale.VaryingDataTypeValue, len(digestsToApply))

	for idx, consensusDigest := range digestsToApply {
		digestAddArgs[idx] = consensusDigest
	}

	err := digest.Add(digestAddArgs...)
	require.NoError(t, err)

	return &types.Header{
		ParentHash: genesisHeader.Hash(),
		Number:     1,
		Digest:     digest,
	}
}
