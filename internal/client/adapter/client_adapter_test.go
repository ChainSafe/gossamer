// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/client/adapter/mocks"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type Hash = hash.H256
type Hasher = runtime.BlakeTwo256
type Number = uint64
type Extrinsic = runtime.OpaqueExtrinsic
type Header = generic.Header[Number, Hash, Hasher]

var currentHasher = Hasher{}

var header = generic.NewHeader[Number, Hash, Hasher](
	1,
	hash.NewRandomH256(),
	hash.NewRandomH256(),
	hash.NewRandomH256(),
	runtime.Digest{},
)

var blockHash = header.Hash()
var blockNumber = header.Number()

var extrinsics = []Extrinsic{
	runtime.OpaqueExtrinsic{Data: []byte("extrinsic1")},
	runtime.OpaqueExtrinsic{Data: []byte("extrinsic2")},
}

var block = generic.NewBlock[Hasher](header, extrinsics)
var signedBlock = generic.NewSignedBlock(block, runtime.Justifications{})

var blockchainInfo = blockchain.Info[Hash, Number]{
	GenesisHash:     blockHash,
	BestHash:        blockHash,
	BestNumber:      blockNumber,
	FinalizedHash:   blockHash,
	FinalizedNumber: blockNumber,
}

func TestBlockStateImplemented(t *testing.T) {
	var _ state.BlockState = &ClientAdapter[Hash, Hasher, Number, Extrinsic, Header]{}
}

func setupTest(t *testing.T) (
	*mocks.Client[Hash, Hasher, Number, Extrinsic, Header],
	*mocks.ClientAdapterDB,
	*mocks.Backend[Hash, Hasher],
	*ClientAdapter[Hash, Hasher, Number, Extrinsic, Header],
) {
	client := mocks.NewClient[Hash, Hasher, Number, Extrinsic, Header](t)
	db := mocks.NewClientAdapterDB(t)
	backend := mocks.NewBackend[Hash, Hasher](t)
	adapter := NewClientAdapter(client, db, backend)

	return client, db, backend, adapter
}

func TestGenesisHash(t *testing.T) {
	client, _, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash := adapter.GenesisHash()
	require.Equal(t, blockchainInfo.GenesisHash.Bytes(), hash.ToBytes())
}

func TestBestNumber(t *testing.T) {
	client, _, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchain.Info[Hash, Number]{
		BestNumber: blockNumber,
	})

	bestNumber, err := adapter.BestBlockNumber()
	require.NoError(t, err)
	require.Equal(t, blockNumber, Number(bestNumber))
}

func TestBestBlockHash(t *testing.T) {
	client, _, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash := adapter.BestBlockHash()
	require.Equal(t, blockchainInfo.BestHash.Bytes(), hash.ToBytes())
}

func TestHasFinalisedBlock(t *testing.T) {
	round := uint64(1)
	setId := uint64(1)
	t.Run("not_finalised_block", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(false, nil)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("finalised_block", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(true, nil)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("error", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(false, expectedError)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.Error(t, err, expectedError)
		require.False(t, has)
	})
}

func TestBlockOps(t *testing.T) {
	t.Run("block_return_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Block(blockchainInfo.BestHash).Return(nil, expectedError)

		t.Run("best_block_error", func(t *testing.T) {
			block, err := adapter.BestBlock()
			require.Error(t, err, expectedError)
			require.Nil(t, block)
		})

		t.Run("best_block_header_error", func(t *testing.T) {
			header, err := adapter.BestBlockHeader()
			require.Error(t, err, expectedError)
			require.Nil(t, header)
		})

		t.Run("block_body_error", func(t *testing.T) {
			body, err := adapter.GetBlockBody(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.Error(t, err, expectedError)
			require.Nil(t, body)
		})

		t.Run("state_root_error", func(t *testing.T) {
			stateRoot, err := adapter.GetBlockStateRoot(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.Error(t, err, expectedError)
			require.Equal(t, common.EmptyHash, stateRoot)
		})
	})

	t.Run("nil_signed_block", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Block(blockchainInfo.BestHash).Return(nil, nil)

		t.Run("best_block_nil", func(t *testing.T) {
			block, err := adapter.BestBlock()
			require.NoError(t, err)
			require.Nil(t, block)
		})

		t.Run("best_block_body_nil", func(t *testing.T) {
			block, err := adapter.GetBlockBody(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.NoError(t, err)
			require.Nil(t, block)
		})

		t.Run("best_block_header_nil", func(t *testing.T) {
			header, err := adapter.BestBlockHeader()
			require.NoError(t, err)
			require.Nil(t, header)
		})

		t.Run("state_root_nil", func(t *testing.T) {
			stateRoot, err := adapter.GetBlockStateRoot(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.NoError(t, err)
			require.Equal(t, common.EmptyHash, stateRoot)
		})
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		t.Run("best_block_ok", func(t *testing.T) {
			expectedBlock, err := types.NewBlockFromGeneric(signedBlock.Block)
			require.NoError(t, err)

			client.EXPECT().Info().Return(blockchainInfo)
			client.EXPECT().Block(blockchainInfo.BestHash).Return(signedBlock, nil)

			block, err := adapter.BestBlock()
			require.NoError(t, err)
			require.NotNil(t, block)

			require.Equal(t, expectedBlock, block)
		})

		t.Run("best_block_header_ok", func(t *testing.T) {
			expectedHeader, err := types.NewHeaderFromGeneric[Number, Hash](header)
			require.NoError(t, err)

			header, err := adapter.BestBlockHeader()
			require.NoError(t, err)
			require.NotNil(t, header)

			require.Equal(t, expectedHeader, header)
		})

		t.Run("block_body_ok", func(t *testing.T) {
			expectedBody := types.NewBodyFromGeneric(extrinsics)
			body, err := adapter.GetBlockBody(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.NoError(t, err)
			require.NotNil(t, body)
			require.Equal(t, expectedBody, *body)
		})

		t.Run("block_state_root_ok", func(t *testing.T) {
			stateRoot, err := adapter.GetBlockStateRoot(common.NewHashFromGeneric(blockchainInfo.BestHash))
			require.NoError(t, err)
			require.NotNil(t, stateRoot)
			require.Equal(t, header.StateRoot().Bytes(), stateRoot.ToBytes())
		})

	})
}

func TestGestJustification(t *testing.T) {
	hash := common.Hash{0x01, 0x02, 0x03}

	t.Run("ok", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		expectedJustification := []byte("justification")
		db.EXPECT().Get(prefixKey(hash, state.JustificationPrefix)).Return(expectedJustification, nil)

		justification, err := adapter.GetJustification(hash)
		require.NoError(t, err)
		require.Equal(t, expectedJustification, justification)
	})

	t.Run("error", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		db.EXPECT().Get(prefixKey(hash, state.JustificationPrefix)).Return(nil, expectedError)

		justification, err := adapter.GetJustification(hash)
		require.Error(t, err, expectedError)
		require.Nil(t, justification)
	})
}

func TestGetBlockByNumber(t *testing.T) {
	t.Run("block_hash_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(nil, expectedError)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, returnedBlock)
	})

	t.Run("block_hash_nil", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(nil, nil)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("get_block_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.Error(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("get_block_returns_nil", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, nil)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.NotNil(t, block)

		expectedBlock, err := types.NewBlockFromGeneric(block)
		require.NoError(t, err)

		require.Equal(t, expectedBlock, returnedBlock)
	})
}

func TestGetHashByNumber(t *testing.T) {
	number := uint(1)
	blockHash := currentHasher.Hash([]byte("blockhash"))

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(Number(number)).Return(&blockHash, nil)

		returnedHash, err := adapter.GetHashByNumber(number)
		require.NoError(t, err)
		require.Equal(t, blockHash.Bytes(), returnedHash.ToBytes())
	})

	t.Run("error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(Number(number)).Return(nil, expectedError)

		returnedHash, err := adapter.GetHashByNumber(number)
		require.Error(t, err, expectedError)
		require.Equal(t, common.EmptyHash, returnedHash)
	})
}

func TestGetHeader(t *testing.T) {
	t.Run("error_getting_block", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		header, err := adapter.GetHeader(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("nil_block", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(nil, nil)

		header, err := adapter.GetHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		expectedHeader, err := types.NewHeaderFromGeneric[Number, Hash](header)
		require.NoError(t, err)

		header, err := adapter.GetHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.NotNil(t, header)
		require.Equal(t, expectedHeader, header)
	})
}

func TestGetHeaderByNumber(t *testing.T) {

	t.Run("block_hash_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, expectedError)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("block_hash_nil", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(nil, nil)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("get_block_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, expectedError)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("get_block_nil", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, nil)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		expectedHeader, err := types.NewHeaderFromGeneric(header)
		require.NoError(t, err)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.NotNil(t, header)
		require.Equal(t, expectedHeader, header)
	})
}

func TestGetHighestFinalizedHeader(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		header, err := adapter.GetHighestFinalisedHeader()
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("nil_header", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, nil)

		header, err := adapter.GetHighestFinalisedHeader()
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(header, nil)

		expectedHeader, err := types.NewHeaderFromGeneric(header)
		require.NoError(t, err)

		header, err := adapter.GetHighestFinalisedHeader()
		require.NoError(t, err)
		require.Equal(t, expectedHeader, header)
	})
}

func TestGetHighestFinalisedHash(t *testing.T) {
	client, _, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash, err := adapter.GetHighestFinalisedHash()
	require.NoError(t, err)
	require.Equal(t, blockchainInfo.FinalizedHash.Bytes(), hash.ToBytes())
}

func TestGetHighestRoundAndSetID(t *testing.T) {

	t.Run("err", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		expectedRound := uint64(0)
		expectedSetID := uint64(0)

		expectedError := errors.New("error")
		db.EXPECT().Get(state.HighestRoundAndSetIDKey).Return(nil, expectedError)

		round, setID, err := adapter.GetHighestRoundAndSetID()
		require.Error(t, err, expectedError)
		require.Equal(t, expectedRound, round)
		require.Equal(t, expectedSetID, setID)
	})
	t.Run("ok", func(t *testing.T) {
		_, db, _, adapter := setupTest(t)

		expectedRound := uint64(10)
		expectedSetID := uint64(20)

		db.EXPECT().Get(state.HighestRoundAndSetIDKey).Return(state.RoundAndSetIDToBytes(expectedRound, expectedSetID), nil)

		round, setID, err := adapter.GetHighestRoundAndSetID()
		require.NoError(t, err)
		require.Equal(t, expectedRound, round)
		require.Equal(t, expectedSetID, setID)
	})
}

func TestGetNonFinalisedBlocks(t *testing.T) {
	t.Run("children_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		blocks := adapter.GetNonFinalisedBlocks()
		require.Len(t, blocks, 0)
	})
	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		children := []Hash{
			hash.NewRandomH256(),
			hash.NewRandomH256(),
		}

		expectedBlocks := []common.Hash{
			common.NewHashFromGeneric(children[0]),
			common.NewHashFromGeneric(children[1]),
		}

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return(children, nil)

		blocks := adapter.GetNonFinalisedBlocks()
		require.Equal(t, expectedBlocks, blocks)
	})
}

func TestHasHeaderInDatabase(t *testing.T) {
	t.Run("get_block_error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.False(t, has)
	})

	t.Run("get_block_nil", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(nil, nil)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})
}

func TestNumberIsFinalised(t *testing.T) {
	t.Run("previous_block_number_finalized", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber) - 1)
		require.NoError(t, err)
		require.True(t, finalised)
	})

	t.Run("same_block_number_finalized", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber))
		require.NoError(t, err)
		require.True(t, finalised)
	})

	t.Run("next_block_number_not_finalized", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber) + 1)
		require.NoError(t, err)
		require.False(t, finalised)
	})
}

func TestHasHeader(t *testing.T) {
	t.Run("header_is_not_finalized", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		children := []Hash{
			blockHash,
		}

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return(children, nil)

		has, err := adapter.HasHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})
	t.Run("header_is_finalized", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return([]Hash{}, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		has, err := adapter.HasHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("not_has_header", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return([]Hash{}, nil)
		client.EXPECT().Block(blockHash).Return(nil, nil)

		has, err := adapter.HasHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.False(t, has)
	})
}

func TestHasJustification(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Justifications(blockHash).Return(nil, expectedError)

		has, err := adapter.HasJustification(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.False(t, has)
	})
	t.Run("has_not_justification", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Justifications(blockHash).Return(runtime.Justifications{}, nil)

		has, err := adapter.HasJustification(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.False(t, has)
	})
	t.Run("has_justification", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		client.EXPECT().Justifications(blockHash).Return(runtime.Justifications{
			runtime.Justification{
				ConsensusEngineID:    [4]byte{'F', 'R', 'N', 'K'},
				EncodedJustification: []byte("justification"),
			},
		}, nil)

		has, err := adapter.HasJustification(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})
}

func TestAddBlock_MissingParams(t *testing.T) {
	t.Run("missing_overlayed_changes", func(t *testing.T) {
		_, _, _, adapter := setupTest(t)

		block := &types.Block{}
		storageVersion := storage.StateVersionV1

		err := adapter.AddBlock(block, nil, &storageVersion)
		require.ErrorIs(t, err, ErrMissingOverlayedChanges)
	})

	t.Run("missing_storage_version", func(t *testing.T) {
		_, _, _, adapter := setupTest(t)

		block := &types.Block{}
		changes := overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()

		err := adapter.AddBlock(block, changes, nil)
		require.ErrorIs(t, err, ErrMissingStorageVersion)
	})
}

func TestAddBlock_Works(t *testing.T) {
	client, _, backend, adapter := setupTest(t)

	block, err := types.NewBlockFromGeneric(block)
	require.NoError(t, err)

	changes := overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	storageVersion := storage.StateVersionV1

	backend.EXPECT().FullStorageRoot(
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(
		hash.NewH256(),
		statemachine.BackendTransaction[hash.H256, runtime.BlakeTwo256]{},
	)
	client.EXPECT().ImportBlock(mock.Anything).Return(nil, nil)

	err = adapter.AddBlock(block, changes, &storageVersion)
	require.NoError(t, err)
}

func TestAddBlock_ImportBlockError(t *testing.T) {
	client, _, backend, adapter := setupTest(t)

	block, err := types.NewBlockFromGeneric(block)
	require.NoError(t, err)

	changes := overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256]()
	storageVersion := storage.StateVersionV1

	expectedError := errors.New("error")

	backend.EXPECT().FullStorageRoot(
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(
		hash.NewH256(),
		statemachine.BackendTransaction[hash.H256, runtime.BlakeTwo256]{},
	)
	client.EXPECT().ImportBlock(mock.Anything).Return(nil, expectedError)

	err = adapter.AddBlock(block, changes, &storageVersion)
	require.ErrorIs(t, err, expectedError)
}

func TestRangeAndRangeInMemory(t *testing.T) {
	hasher := *new(Hasher)
	fromHash := common.MustBlake2bHash([]byte("fromHash"))
	toHash := common.MustBlake2bHash([]byte("toHash"))

	t.Run("no_from_header_metadata", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")

		client.EXPECT().HeaderMetadata(hasher.NewHash(fromHash.ToBytes())).Return(
			blockchain.CachedHeaderMetadata[Hash, Number]{},
			expectedError,
		)

		resRange, errRange := adapter.Range(fromHash, toHash)
		resRangeInMemory, errRangeInMemory := adapter.RangeInMemory(fromHash, toHash)

		require.Equal(t, resRange, resRangeInMemory)
		require.Equal(t, errRange, errRangeInMemory)
		require.Error(t, errRange)
		require.Equal(t, expectedError, errRange)
	})

	t.Run("no_to_header_metadata", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		fromMeta := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(fromHash.ToBytes()),
			Number:    Number(100),
			Parent:    hasher.Hash([]byte("fromParent")),
			StateRoot: hasher.Hash([]byte("fromStateRoot")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(fromHash.ToBytes())).Return(fromMeta, nil)

		expectedError := errors.New("error")
		client.EXPECT().HeaderMetadata(hasher.NewHash(toHash.ToBytes())).Return(
			blockchain.CachedHeaderMetadata[Hash, Number]{},
			expectedError,
		)

		resRange, errRange := adapter.Range(fromHash, toHash)
		resRangeInMemory, errRangeInMemory := adapter.RangeInMemory(fromHash, toHash)

		require.Equal(t, resRange, resRangeInMemory)
		require.Equal(t, errRange, errRangeInMemory)
		require.Error(t, errRange)
		require.Equal(t, expectedError, errRange)
	})

	t.Run("ok_same_block", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		sameMeta := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(fromHash.ToBytes()),
			Number:    Number(100),
			Parent:    hasher.Hash([]byte("parent")),
			StateRoot: hasher.Hash([]byte("stateRoot")),
		}

		client.EXPECT().HeaderMetadata(hasher.NewHash(fromHash.ToBytes())).Return(sameMeta, nil)

		resRange, errRange := adapter.Range(fromHash, fromHash)
		resRangeInMemory, errRangeInMemory := adapter.RangeInMemory(fromHash, fromHash)

		require.Equal(t, resRange, resRangeInMemory)
		require.Equal(t, errRange, errRangeInMemory)
		require.NoError(t, errRange)
		require.Len(t, resRange, 1)
		require.Equal(t, common.NewHashFromGeneric(hasher.NewHash(fromHash.ToBytes())), resRange[0])
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		fromMeta := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(fromHash.ToBytes()),
			Number:    Number(100),
			Parent:    hasher.Hash([]byte("fromParent")),
			StateRoot: hasher.Hash([]byte("fromStateRoot")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(fromHash.ToBytes())).Return(fromMeta, nil)

		toMeta1 := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(toHash.ToBytes()),
			Number:    Number(102),
			Parent:    hasher.Hash([]byte("toParent")),
			StateRoot: hasher.Hash([]byte("toStateRoot")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(toHash.ToBytes())).Return(toMeta1, nil)

		toMeta2 := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.Hash([]byte("toParent")),
			Number:    Number(101),
			Parent:    hasher.NewHash(fromHash.ToBytes()),
			StateRoot: hasher.Hash([]byte("toParentStateRoot")),
		}
		client.EXPECT().HeaderMetadata(hasher.Hash([]byte("toParent"))).Return(toMeta2, nil)

		resRange, errRange := adapter.Range(fromHash, toHash)
		resRangeInMemory, errRangeInMemory := adapter.RangeInMemory(fromHash, toHash)

		require.Equal(t, resRange, resRangeInMemory)
		require.Equal(t, errRange, errRangeInMemory)
		require.NoError(t, errRange)
		require.Len(t, resRange, 3)

		// from -> mid -> to
		expectedHashes := []common.Hash{
			common.NewHashFromGeneric(hasher.NewHash(fromHash.ToBytes())),
			common.NewHashFromGeneric(hasher.Hash([]byte("toParent"))),
			common.NewHashFromGeneric(hasher.NewHash(toHash.ToBytes())),
		}
		require.Equal(t, expectedHashes, resRange)
	})
}

func TestLowestCommonAncestor(t *testing.T) {
	hasher := *new(Hasher)
	hashA := common.MustBlake2bHash([]byte("hashA"))
	hashB := common.MustBlake2bHash([]byte("hashB"))

	t.Run("no_header_A_metadata", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().HeaderMetadata(hasher.NewHash(hashA.ToBytes())).Return(
			blockchain.CachedHeaderMetadata[Hash, Number]{},
			expectedError,
		)

		_, err := adapter.LowestCommonAncestor(hashA, hashB)
		require.Error(t, err)
		require.Equal(t, expectedError, err)
	})

	t.Run("no_header_B_metadata", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		header1 := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(hashA.ToBytes()),
			Number:    Number(101),
			Parent:    hasher.Hash([]byte("parentA")),
			StateRoot: hasher.Hash([]byte("stateRoot1")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(hashA.ToBytes())).Return(header1, nil)

		expectedError := errors.New("error")
		client.EXPECT().HeaderMetadata(hasher.NewHash(hashB.ToBytes())).Return(
			blockchain.CachedHeaderMetadata[Hash, Number]{},
			expectedError,
		)

		_, err := adapter.LowestCommonAncestor(hashA, hashB)
		require.Error(t, err)
		require.Equal(t, expectedError, err)
	})

	t.Run("ok_same_block", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		sameHeader := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(hashA.ToBytes()),
			Number:    Number(100),
			Parent:    hasher.Hash([]byte("parent")),
			StateRoot: hasher.Hash([]byte("stateRoot")),
		}

		client.EXPECT().HeaderMetadata(hasher.NewHash(hashA.ToBytes())).Return(sameHeader, nil)

		result, err := adapter.LowestCommonAncestor(hashA, hashA)
		require.NoError(t, err)
		require.Equal(t, hashA, result)
	})

	t.Run("ok_common_ancestor", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		commonAncestorHash := hasher.Hash([]byte("commonAncestor"))

		// left branch
		headerA := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(hashA.ToBytes()),
			Number:    Number(102),
			Parent:    hasher.Hash([]byte("parentA")),
			StateRoot: hasher.Hash([]byte("stateRootA")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(hashA.ToBytes())).Return(headerA, nil)

		// right branch
		headerB := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      hasher.NewHash(hashB.ToBytes()),
			Number:    Number(102),
			Parent:    hasher.Hash([]byte("parentB")),
			StateRoot: hasher.Hash([]byte("stateRootB")),
		}
		client.EXPECT().HeaderMetadata(hasher.NewHash(hashB.ToBytes())).Return(headerB, nil)

		// common ancestor
		commonAncestor := blockchain.CachedHeaderMetadata[Hash, Number]{
			Hash:      commonAncestorHash,
			Number:    Number(100),
			Parent:    hasher.Hash([]byte("grandParent")),
			StateRoot: hasher.Hash([]byte("commonStateRoot")),
		}

		client.EXPECT().HeaderMetadata(hasher.Hash([]byte("parentA"))).Return(commonAncestor, nil)
		client.EXPECT().HeaderMetadata(hasher.Hash([]byte("parentB"))).Return(commonAncestor, nil)

		client.EXPECT().InsertHeaderMetadata(mock.Anything, mock.Anything)

		result, err := adapter.LowestCommonAncestor(hashA, hashB)
		require.NoError(t, err)
		require.Equal(t, common.NewHashFromGeneric(commonAncestorHash), result)
	})
}

func TestIsDescendantOf(t *testing.T) {
	hasher := *new(Hasher)
	parentHash := common.MustBlake2bHash([]byte("parent"))
	childHash := common.MustBlake2bHash([]byte("child"))

	t.Run("same_hash_returns_false", func(t *testing.T) {
		_, _, _, adapter := setupTest(t)

		result, err := adapter.IsDescendantOf(parentHash, parentHash)
		require.NoError(t, err)
		require.False(t, result)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		parentHash := hasher.NewHash(parentHash.ToBytes())
		childHash := hasher.NewHash(childHash.ToBytes())

		childHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(101),
			childHash,
			hasher.Hash([]byte("childStateRoot")),
			parentHash,
			runtime.Digest{},
		)

		childCachedHeader := blockchain.NewCachedHeaderMetadata(childHeader)
		client.EXPECT().HeaderMetadata(childHash).Return(childCachedHeader, nil)

		result, err := adapter.IsDescendantOf(common.NewHashFromGeneric(parentHash), common.NewHashFromGeneric(childHash))
		require.NoError(t, err)
		require.True(t, result)
	})

	t.Run("ok_long_chain", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		parentHash := hasher.NewHash(parentHash.ToBytes())
		childHash := hasher.NewHash(childHash.ToBytes())

		grandParentHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(99),
			hasher.NewHash([]byte("grandParentHash")),
			hasher.Hash([]byte("grandParentStateRoot")),
			hasher.Hash([]byte("greatGrandParent")),
			runtime.Digest{},
		)

		parentHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(100),
			parentHash,
			hasher.Hash([]byte("parentStateRoot")),
			grandParentHeader.Hash(),
			runtime.Digest{},
		)
		parentCachedHeader := blockchain.NewCachedHeaderMetadata(parentHeader)
		client.EXPECT().HeaderMetadata(parentHeader.Hash()).Return(parentCachedHeader, nil)

		intermediateHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(101),
			hasher.NewHash([]byte("intermediateHash")),
			hasher.Hash([]byte("intermediateStateRoot")),
			parentHeader.Hash(),
			runtime.Digest{},
		)
		intermediateCachedHeader := blockchain.NewCachedHeaderMetadata(intermediateHeader)
		client.EXPECT().HeaderMetadata(intermediateHeader.Hash()).Return(intermediateCachedHeader, nil)

		childHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(102),
			childHash,
			hasher.Hash([]byte("childStateRoot")),
			intermediateHeader.Hash(),
			runtime.Digest{},
		)

		childCachedHeader := blockchain.NewCachedHeaderMetadata(childHeader)
		client.EXPECT().HeaderMetadata(childHeader.Hash()).Return(childCachedHeader, nil)

		client.EXPECT().InsertHeaderMetadata(mock.Anything, mock.Anything)

		result, err := adapter.IsDescendantOf(
			common.NewHashFromGeneric(parentHeader.Hash()),
			common.NewHashFromGeneric(childHeader.Hash()),
		)
		require.NoError(t, err)
		require.True(t, result)
	})

	t.Run("child_is_not_ancestor_of_parent", func(t *testing.T) {
		client, _, _, adapter := setupTest(t)

		parentHash := hasher.NewHash(parentHash.ToBytes())
		childHash := hasher.NewHash(childHash.ToBytes())

		parentHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(100),
			parentHash,
			hasher.Hash([]byte("parentStateRoot")),
			hasher.NewHash([]byte("grandParent")),
			runtime.Digest{},
		)
		parentCachedHeader := blockchain.NewCachedHeaderMetadata(parentHeader)
		client.EXPECT().HeaderMetadata(parentHeader.Hash()).Return(parentCachedHeader, nil)

		childHeader := generic.NewHeader[Number, Hash, Hasher](
			Number(101),
			childHash,
			hasher.Hash([]byte("childStateRoot")),
			parentHeader.Hash(),
			runtime.Digest{},
		)
		childCachedHeader := blockchain.NewCachedHeaderMetadata(childHeader)
		client.EXPECT().HeaderMetadata(childHeader.Hash()).Return(childCachedHeader, nil)

		result, err := adapter.IsDescendantOf(
			common.NewHashFromGeneric(childHeader.Hash()),
			common.NewHashFromGeneric(parentHeader.Hash()),
		)
		require.NoError(t, err)
		require.False(t, result)
	})
}
