// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

var genesisBABEConfig = &types.BabeConfiguration{
	SlotDuration:       1000,
	EpochLength:        200,
	C1:                 1,
	C2:                 4,
	GenesisAuthorities: []types.AuthorityRaw{},
	Randomness:         [32]byte{},
	SecondarySlots:     0,
}

func newEpochStateFromGenesis(t *testing.T) *EpochState {
	db := NewInMemoryDB(t)
	blockState := newTestBlockState(t, newTriesEmpty())
	s, err := NewEpochStateFromGenesis(db, blockState, genesisBABEConfig)
	require.NoError(t, err)
	return s
}

func TestNewEpochStateFromGenesis(t *testing.T) {
	_ = newEpochStateFromGenesis(t)
}

func TestEpochState_CurrentEpoch(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	epoch, err := s.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(0), epoch)

	err = s.StoreCurrentEpoch(1)
	require.NoError(t, err)
	epoch, err = s.GetCurrentEpoch()
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)
}

func TestEpochState_EpochData(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	auth := types.AuthorityRaw{
		Key:    keyring.Alice().Public().(*sr25519.PublicKey).AsBytes(),
		Weight: 1,
	}

	info := &types.EpochDataRaw{
		Authorities: []types.AuthorityRaw{auth},
		Randomness:  [32]byte{77},
	}

	err = s.StoreEpochDataRaw(1, info)
	require.NoError(t, err)
	res, err := s.GetEpochDataRaw(1, nil)
	require.NoError(t, err)
	require.Equal(t, info.Randomness, res.Randomness)

	for i, auth := range res.Authorities {
		require.Equal(t, info.Authorities[i], auth)
	}
}

func TestEpochState_GetStartSlotForEpoch(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	// let's say first slot is 1 second after January 1, 1970 UTC
	startAtTime := time.Unix(1, 0)
	slotDuration := time.Millisecond * time.Duration(genesisBABEConfig.SlotDuration)
	firstSlot := uint64(startAtTime.UnixNano()) / uint64(slotDuration.Nanoseconds())

	digest := types.NewDigest()
	di, err := types.NewBabeSecondaryPlainPreDigest(0, firstSlot).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest.Add(*di)
	require.NoError(t, err)

	header1 := types.Header{
		Number:     1,
		Digest:     digest,
		ParentHash: s.blockState.genesisHash,
	}

	err = s.blockState.AddBlock(&types.Block{
		Header: header1,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	start, err := s.GetStartSlotForEpoch(0, header1.Hash())
	fmt.Println(start)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)

	start, err = s.GetStartSlotForEpoch(1, header1.Hash())
	fmt.Println(start)
	require.NoError(t, err)
	require.Equal(t, uint64(201), start)

	start, err = s.GetStartSlotForEpoch(2, header1.Hash())
	fmt.Println(start)
	require.NoError(t, err)
	require.Equal(t, uint64(401), start)
}

func TestEpochState_ConfigData(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	data := &types.ConfigData{
		C1:             1,
		C2:             8,
		SecondarySlots: 1,
	}

	err := s.StoreConfigData(1, data)
	require.NoError(t, err)

	ret, err := s.GetConfigData(1, nil)
	require.NoError(t, err)
	require.Equal(t, data, ret)

	ret, err = s.GetLatestConfigData()
	require.NoError(t, err)
	require.Equal(t, data, ret)
}

func createAndImportBlockOne(t *testing.T, slotNumber uint64, blockState *BlockState) (blockOneHeader *types.Header) {
	babeHeader := types.NewBabeDigest()
	err := babeHeader.SetValue(*types.NewBabePrimaryPreDigest(0, slotNumber, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	enc, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	d := types.NewBABEPreRuntimeDigest(enc)
	digest := types.NewDigest()
	digest.Add(*d)

	blockOneHeader = &types.Header{
		Number:     1,
		Digest:     digest,
		ParentHash: blockState.genesisHash,
	}

	err = blockState.AddBlock(&types.Block{
		Header: *blockOneHeader,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	return blockOneHeader
}

func TestEpochState_GetEpochForBlock(t *testing.T) {
	s := newEpochStateFromGenesis(t)

	firstSlot := uint64(1)
	blockOneHeader := createAndImportBlockOne(t, firstSlot, s.blockState)

	babeHeader := types.NewBabeDigest()
	err := babeHeader.SetValue(*types.NewBabePrimaryPreDigest(0, s.epochLength*1+1, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	enc, err := scale.Marshal(babeHeader)
	require.NoError(t, err)
	d := types.NewBABEPreRuntimeDigest(enc)
	digest := types.NewDigest()
	digest.Add(*d)

	header2 := &types.Header{
		Number:     2,
		Digest:     digest,
		ParentHash: blockOneHeader.Hash(),
	}

	err = s.blockState.AddBlock(&types.Block{
		Header: *header2,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	epoch, err := s.GetEpochForBlock(header2)
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	babeHeader = types.NewBabeDigest()
	err = babeHeader.SetValue(*types.NewBabePrimaryPreDigest(0, s.epochLength*2+1, [32]byte{}, [64]byte{}))
	require.NoError(t, err)
	enc, err = scale.Marshal(babeHeader)
	require.NoError(t, err)
	d = types.NewBABEPreRuntimeDigest(enc)
	digest2 := types.NewDigest()
	digest2.Add(*d)

	header3 := &types.Header{
		Number:     3,
		Digest:     digest2,
		ParentHash: header2.Hash(),
	}

	err = s.blockState.AddBlock(&types.Block{
		Header: *header3,
		Body:   *types.NewBody([]types.Extrinsic{}),
	})
	require.NoError(t, err)

	epoch, err = s.GetEpochForBlock(header3)
	require.NoError(t, err)
	require.Equal(t, uint64(2), epoch)
}

func TestEpochState_SetAndGetSlotDuration(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	expected := time.Millisecond * time.Duration(genesisBABEConfig.SlotDuration)

	ret, err := s.GetSlotDuration()
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func TestEpochState_GetEpochFromTime(t *testing.T) {
	s := newEpochStateFromGenesis(t)
	s.blockState = newTestBlockState(t, newTriesEmpty())

	// let's say first slot is 1 second after January 1, 1970 UTC
	start := time.Unix(1, 0)
	slotDuration := time.Millisecond * time.Duration(genesisBABEConfig.SlotDuration)
	firstSlot := uint64(start.UnixNano()) / uint64(slotDuration.Nanoseconds())

	digest := types.NewDigest()
	di, err := types.NewBabeSecondaryPlainPreDigest(0, firstSlot).ToPreRuntimeDigest()
	require.NoError(t, err)
	require.NotNil(t, di)
	err = digest.Add(*di)
	require.NoError(t, err)

	header1 := types.Header{
		Number:     1,
		Digest:     digest,
		ParentHash: s.blockState.genesisHash,
	}

	err = s.blockState.AddBlock(&types.Block{
		Header: header1,
		Body:   types.Body{},
	})
	require.NoError(t, err)

	epochDuration, err := time.ParseDuration(
		fmt.Sprintf("%dms",
			genesisBABEConfig.SlotDuration*genesisBABEConfig.EpochLength))
	require.NoError(t, err)

	require.NoError(t, err)

	epoch, err := s.GetEpochFromTime(start, header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(0), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration), header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration/2), header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(0), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration*3/2), header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration*100+1), header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(100), epoch)

	epoch, err = s.GetEpochFromTime(start.Add(epochDuration*100-1), header1.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(99), epoch)
}

type inMemoryBABEData[T any] struct {
	epoch    uint64
	hashes   []common.Hash
	nextData []T
}

func TestStoreAndFinalizeBabeNextEpochData(t *testing.T) {
	/*
	* Setup the services: StateService, DigestHandler, EpochState
	* and VerificationManager
	 */

	keyring, _ := keystore.NewSr25519Keyring()
	keyPairs := []*sr25519.Keypair{
		keyring.KeyAlice, keyring.KeyBob, keyring.KeyCharlie,
		keyring.KeyDave, keyring.KeyEve, keyring.KeyFerdie,
		keyring.KeyGeorge, keyring.KeyHeather, keyring.KeyIan,
	}

	authorities := make([]types.AuthorityRaw, len(keyPairs))
	for i, keyPair := range keyPairs {
		authorities[i] = types.AuthorityRaw{
			Key: keyPair.Public().(*sr25519.PublicKey).AsBytes(),
		}
	}

	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 301, // block on epoch 0 with digest for epoch 1
		VRFOutput:  [32]byte{},
		VRFProof:   [64]byte{},
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := types.NewDigest()

	require.NoError(t, digest.Add(*preRuntimeDigest))

	// a random finalized header for testing purposes
	finalizedHeader := &types.Header{
		ParentHash: common.Hash{},
		Number:     1,
		Digest:     digest,
	}

	finalizedHeaderHash := finalizedHeader.Hash()

	tests := map[string]struct {
		finalizedHeader      *types.Header
		inMemoryEpoch        []inMemoryBABEData[types.NextEpochData]
		finalizeEpoch        uint64
		expectErr            error
		shouldRemainInMemory int
	}{
		"store_and_finalize_successfully": {
			shouldRemainInMemory: 2,
			finalizeEpoch:        1,
			finalizedHeader:      finalizedHeader,
			inMemoryEpoch: []inMemoryBABEData[types.NextEpochData]{
				{
					epoch: 1,
					hashes: []common.Hash{
						common.MustHexToHash("0x9da3ce2785da743bfbc13449db7dcb7a69c07ca914276d839abe7bedc6ac8fed"),
						common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
						finalizedHeaderHash,
					},
					nextData: []types.NextEpochData{
						{
							Authorities: authorities[:3],
							Randomness:  [32]byte{1},
						},
						{
							Authorities: authorities[3:6],
							Randomness:  [32]byte{2},
						},
						{
							Authorities: authorities[6:],
							Randomness:  [32]byte{3},
						},
					},
				},
				{
					epoch: 2,
					hashes: []common.Hash{
						common.MustHexToHash("0x5b940c7fc0a1c5a58e4d80c5091dd003303b8f18e90a989f010c1be6f392bed1"),
						common.MustHexToHash("0xd380bee22de487a707cbda65dd9d4e2188f736908c42cf390c8919d4f7fc547c"),
					},
					nextData: []types.NextEpochData{
						{
							Authorities: authorities[6:],
							Randomness:  [32]byte{1},
						},
						{
							Authorities: authorities[:3],
							Randomness:  [32]byte{2},
						},
						{
							Authorities: authorities[3:6],
							Randomness:  [32]byte{3},
						},
					},
				},
				{
					epoch: 3,
					hashes: []common.Hash{
						common.MustHexToHash("0xab5c9230a7dde8bb90a6728ba4a0165423294dac14336b1443f865b796ff682c"),
					},
					nextData: []types.NextEpochData{
						{
							Authorities: authorities[6:],
							Randomness:  [32]byte{1},
						},
					},
				},
			},
		},
		"cannot_finalize_hash_not_stored": {
			shouldRemainInMemory: 1,
			finalizeEpoch:        1,
			// this header hash is not in the database
			finalizedHeader: finalizedHeader,
			expectErr:       errHashNotPersisted,
			inMemoryEpoch: []inMemoryBABEData[types.NextEpochData]{
				{
					epoch: 1,
					hashes: []common.Hash{
						common.MustHexToHash("0x9da3ce2785da743bfbc13449db7dcb7a69c07ca914276d839abe7bedc6ac8fed"),
						common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
						common.MustHexToHash("0xc0096358534ec8d21d01d34b836eed476a1c343f8724fa2153dc0725ad797a90"),
					},
					nextData: []types.NextEpochData{
						{
							Authorities: authorities[:3],
							Randomness:  [32]byte{1},
						},
						{
							Authorities: authorities[3:6],
							Randomness:  [32]byte{2},
						},
						{
							Authorities: authorities[6:],
							Randomness:  [32]byte{3},
						},
					},
				},
			},
		},
		"cannot_finalize_in_memory_epoch_not_found": {
			shouldRemainInMemory: 0,
			finalizeEpoch:        3, // try to finalize a epoch that does not exists
			finalizedHeader:      finalizedHeader,
			expectErr:            ErrEpochNotInMemory,
			inMemoryEpoch:        []inMemoryBABEData[types.NextEpochData]{},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			epochState := newEpochStateFromGenesis(t)

			for _, e := range tt.inMemoryEpoch {
				for i, hash := range e.hashes {
					epochState.storeBABENextEpochData(e.epoch, hash, e.nextData[i])
				}
			}

			require.Len(t, epochState.nextEpochData, len(tt.inMemoryEpoch))
			expectedNextEpochData := epochState.nextEpochData[tt.finalizeEpoch][tt.finalizedHeader.Hash()]

			err := epochState.blockState.SetHeader(tt.finalizedHeader)
			require.NoError(t, err)

			err = epochState.FinalizeBABENextEpochData(tt.finalizedHeader)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)

				expected := expectedNextEpochData.ToEpochDataRaw()
				gotNextEpochData, err := epochState.GetEpochDataRaw(tt.finalizeEpoch, nil)
				require.NoError(t, err)

				require.Equal(t, expected, gotNextEpochData)
			}

			// should delete previous epochs since the most up to date epoch is stored
			require.Len(t, epochState.nextEpochData, tt.shouldRemainInMemory)
		})
	}
}

func newBlockWithPrimaryDigest(t *testing.T, slotNumber uint64, blockNumber uint) *types.Header {
	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: slotNumber, // block on epoch 0 with changes to epoch 1
		VRFOutput:  [32]byte{},
		VRFProof:   [64]byte{},
	}

	preRuntimeDigest, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	require.NoError(t, err)

	digest := types.NewDigest()

	require.NoError(t, digest.Add(*preRuntimeDigest))

	return &types.Header{
		ParentHash: common.Hash{},
		Number:     blockNumber,
		Digest:     digest,
	}
}

func TestStoreAndFinalizeBabeNextConfigData(t *testing.T) {
	chainFirstSlotNumber := uint64(1)
	blockNumber1 := newBlockWithPrimaryDigest(t,
		chainFirstSlotNumber, 1)
	blockNumber2 := newBlockWithPrimaryDigest(t,
		chainFirstSlotNumber+genesisBABEConfig.EpochLength, 2)

	finalizedHeaders := []*types.Header{blockNumber1, blockNumber2}

	tests := map[string]struct {
		finalizedHeader      *types.Header
		inMemoryEpoch        []inMemoryBABEData[types.NextConfigDataV1]
		finalizedEpoch       uint64
		expectErr            error
		shouldRemainInMemory int
	}{
		"store_and_finalize_successfully": {
			shouldRemainInMemory: 1,
			finalizedEpoch:       2,
			finalizedHeader:      blockNumber2,
			inMemoryEpoch: []inMemoryBABEData[types.NextConfigDataV1]{
				{
					epoch: 1,
					hashes: []common.Hash{
						common.MustHexToHash("0x9da3ce2785da743bfbc13449db7dcb7a69c07ca914276d839abe7bedc6ac8fed"),
						common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
						common.MustHexToHash("0xc0096358534ec8d21d01d34b836eed476a1c343f8724fa2153dc0725ad797a90"),
					},
					nextData: []types.NextConfigDataV1{
						{
							C1:             1,
							C2:             2,
							SecondarySlots: 0,
						},
						{
							C1:             2,
							C2:             3,
							SecondarySlots: 1,
						},
						{
							C1:             3,
							C2:             4,
							SecondarySlots: 0,
						},
					},
				},
				{
					epoch: 2,
					hashes: []common.Hash{
						common.MustHexToHash("0x5b940c7fc0a1c5a58e4d80c5091dd003303b8f18e90a989f010c1be6f392bed1"),
						common.MustHexToHash("0xd380bee22de487a707cbda65dd9d4e2188f736908c42cf390c8919d4f7fc547c"),
						blockNumber2.Hash(),
					},
					nextData: []types.NextConfigDataV1{
						{
							C1:             1,
							C2:             2,
							SecondarySlots: 0,
						},
						{
							C1:             2,
							C2:             3,
							SecondarySlots: 1,
						},
						{
							C1:             3,
							C2:             4,
							SecondarySlots: 0,
						},
					},
				},
				{
					epoch: 3,
					hashes: []common.Hash{
						common.MustHexToHash("0xab5c9230a7dde8bb90a6728ba4a0165423294dac14336b1443f865b796ff682c"),
					},
					nextData: []types.NextConfigDataV1{
						{
							C1:             1,
							C2:             2,
							SecondarySlots: 0,
						},
					},
				},
			},
		},
		"cannot_finalize_hash_doesnt_exists": {
			shouldRemainInMemory: 1,
			finalizedEpoch:       2,
			finalizedHeader:      blockNumber2, // finalize when the hash does not exist
			expectErr:            errHashNotPersisted,
			inMemoryEpoch: []inMemoryBABEData[types.NextConfigDataV1]{
				{
					epoch: 2,
					hashes: []common.Hash{
						common.MustHexToHash("0x9da3ce2785da743bfbc13449db7dcb7a69c07ca914276d839abe7bedc6ac8fed"),
						common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
						common.MustHexToHash("0xc0096358534ec8d21d01d34b836eed476a1c343f8724fa2153dc0725ad797a90"),
					},
					nextData: []types.NextConfigDataV1{
						{
							C1:             1,
							C2:             2,
							SecondarySlots: 0,
						},
						{
							C1:             2,
							C2:             3,
							SecondarySlots: 1,
						},
						{
							C1:             3,
							C2:             4,
							SecondarySlots: 0,
						},
					},
				},
			},
		},
		"in_memory_config_not_found_shouldnt_return_error": {
			shouldRemainInMemory: 0,
			finalizedEpoch:       1, // try to finalize an epoch that does not exist
			finalizedHeader:      blockNumber1,
			inMemoryEpoch:        []inMemoryBABEData[types.NextConfigDataV1]{},
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			epochState := newEpochStateFromGenesis(t)

			for _, finalized := range finalizedHeaders {
				// mapping number #1 to the block hash
				// then we can retrieve the slot number
				// using the block number
				err := epochState.blockState.db.Put(
					headerHashKey(uint64(finalized.Number)),
					finalized.Hash().ToBytes(),
				)
				require.NoError(t, err)

				err = epochState.blockState.SetHeader(finalized)
				require.NoError(t, err)
			}

			for _, e := range tt.inMemoryEpoch {
				for i, hash := range e.hashes {
					epochState.storeBABENextConfigData(e.epoch, hash, e.nextData[i])
				}
			}

			require.Len(t, epochState.nextConfigData, len(tt.inMemoryEpoch))

			// if there is no data in memory we try to finalize the next config data
			// it should return nil since next epoch config data will not be in every epoch's first block
			if len(tt.inMemoryEpoch) == 0 {
				err := epochState.FinalizeBABENextConfigData(tt.finalizedHeader)
				require.NoError(t, err)
				return
			}

			expectedConfigData := epochState.nextConfigData[tt.finalizedEpoch][tt.finalizedHeader.Hash()]

			err := epochState.FinalizeBABENextConfigData(tt.finalizedHeader)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)

				gotConfigData, err := epochState.GetConfigData(tt.finalizedEpoch, nil)
				require.NoError(t, err)
				require.Equal(t, expectedConfigData.ToConfigData(), gotConfigData)
			}

			// should delete previous epochs since the most up to date epoch is stored
			require.Len(t, epochState.nextConfigData, tt.shouldRemainInMemory)
		})
	}
}
