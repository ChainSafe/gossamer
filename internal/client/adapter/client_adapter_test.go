// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/client/adapter/mocks"
	"github.com/ChainSafe/gossamer/internal/client/db/offchain"
	memorykvdb "github.com/ChainSafe/gossamer/internal/kvdb/memory-kvdb"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	offchainapi "github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

type Hash = hash.H256
type Hasher = runtime.BlakeTwo256
type Number = uint64
type Extrinsic = runtime.OpaqueExtrinsic
type Header = *generic.Header[Number, Hash, Hasher]

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
	*ClientAdapter[Hash, Hasher, Number, Extrinsic, Header],
) {
	client := mocks.NewClient[Hash, Hasher, Number, Extrinsic, Header](t)
	db := mocks.NewClientAdapterDB(t)
	adapter := NewClientAdapter(client, db)

	return client, db, adapter
}

func TestGenesisHash(t *testing.T) {
	client, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash := adapter.GenesisHash()
	require.Equal(t, blockchainInfo.GenesisHash.Bytes(), hash.ToBytes())
}

func TestBestNumber(t *testing.T) {
	client, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchain.Info[Hash, Number]{
		BestNumber: blockNumber,
	})

	bestNumber, err := adapter.BestBlockNumber()
	require.NoError(t, err)
	require.Equal(t, blockNumber, Number(bestNumber))
}

func TestBestBlockHash(t *testing.T) {
	client, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash := adapter.BestBlockHash()
	require.Equal(t, blockchainInfo.BestHash.Bytes(), hash.ToBytes())
}

func TestHasFinalisedBlock(t *testing.T) {
	round := uint64(1)
	setId := uint64(1)
	t.Run("not_finalised_block", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(false, nil)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("finalised_block", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(true, nil)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("error", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		expectedError := errors.New("error")
		db.EXPECT().Has(state.FinalisedHashKey(round, setId)).Return(false, expectedError)

		has, err := adapter.HasFinalisedBlock(round, setId)
		require.Error(t, err, expectedError)
		require.False(t, has)
	})
}

func TestBlockOps(t *testing.T) {
	t.Run("block_return_error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

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
		_, db, adapter := setupTest(t)

		expectedJustification := []byte("justification")
		db.EXPECT().Get(prefixKey(hash, state.JustificationPrefix)).Return(expectedJustification, nil)

		justification, err := adapter.GetJustification(hash)
		require.NoError(t, err)
		require.Equal(t, expectedJustification, justification)
	})

	t.Run("error", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		expectedError := errors.New("error")
		db.EXPECT().Get(prefixKey(hash, state.JustificationPrefix)).Return(nil, expectedError)

		justification, err := adapter.GetJustification(hash)
		require.Error(t, err, expectedError)
		require.Nil(t, justification)
	})
}

func TestGetBlockByNumber(t *testing.T) {
	t.Run("block_hash_error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(nil, expectedError)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, returnedBlock)
	})

	t.Run("block_hash_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(nil, nil)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("get_block_error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.Error(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("get_block_returns_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, nil)

		returnedBlock, err := adapter.GetBlockByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, returnedBlock)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(Number(number)).Return(&blockHash, nil)

		returnedHash, err := adapter.GetHashByNumber(number)
		require.NoError(t, err)
		require.Equal(t, blockHash.Bytes(), returnedHash.ToBytes())
	})

	t.Run("error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(Number(number)).Return(nil, expectedError)

		returnedHash, err := adapter.GetHashByNumber(number)
		require.Error(t, err, expectedError)
		require.Equal(t, common.EmptyHash, returnedHash)
	})
}

func TestGetHeader(t *testing.T) {
	t.Run("error_getting_block", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		header, err := adapter.GetHeader(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("nil_block", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(nil, nil)

		header, err := adapter.GetHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, expectedError)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("block_hash_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(nil, nil)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("get_block_error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, expectedError)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("get_block_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().BlockHash(blockNumber).Return(&blockHash, nil)
		client.EXPECT().Block(blockHash).Return(nil, nil)

		header, err := adapter.GetHeaderByNumber(uint(blockNumber))
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		header, err := adapter.GetHighestFinalisedHeader()
		require.Error(t, err, expectedError)
		require.Nil(t, header)
	})

	t.Run("nil_header", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, nil)

		header, err := adapter.GetHighestFinalisedHeader()
		require.NoError(t, err)
		require.Nil(t, header)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(&header, nil)

		expectedHeader, err := types.NewHeaderFromGeneric(header)
		require.NoError(t, err)

		header, err := adapter.GetHighestFinalisedHeader()
		require.NoError(t, err)
		require.Equal(t, expectedHeader, header)
	})
}

func TestGetHighestFinalisedHash(t *testing.T) {
	client, _, adapter := setupTest(t)

	client.EXPECT().Info().Return(blockchainInfo)

	hash, err := adapter.GetHighestFinalisedHash()
	require.NoError(t, err)
	require.Equal(t, blockchainInfo.FinalizedHash.Bytes(), hash.ToBytes())
}

func TestGetHighestRoundAndSetID(t *testing.T) {

	t.Run("err", func(t *testing.T) {
		_, db, adapter := setupTest(t)

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
		_, db, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		blocks := adapter.GetNonFinalisedBlocks()
		require.Len(t, blocks, 0)
	})
	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Block(blockHash).Return(nil, expectedError)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.False(t, has)
	})

	t.Run("get_block_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(nil, nil)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		has, err := adapter.HasHeaderInDatabase(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})
}

func TestNumberIsFinalised(t *testing.T) {
	t.Run("previous_block_number_finalized", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber) - 1)
		require.NoError(t, err)
		require.True(t, finalised)
	})

	t.Run("same_block_number_finalized", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber))
		require.NoError(t, err)
		require.True(t, finalised)
	})

	t.Run("next_block_number_not_finalized", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)

		finalised, err := adapter.NumberIsFinalised(uint(blockNumber) + 1)
		require.NoError(t, err)
		require.False(t, finalised)
	})
}

func TestHasHeader(t *testing.T) {
	t.Run("header_is_not_finalized", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return([]Hash{}, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		has, err := adapter.HasHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("not_has_header", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Justifications(blockHash).Return(nil, expectedError)

		has, err := adapter.HasJustification(common.NewHashFromGeneric(blockHash))
		require.Error(t, err, expectedError)
		require.False(t, has)
	})
	t.Run("has_not_justification", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Justifications(blockHash).Return(runtime.Justifications{}, nil)

		has, err := adapter.HasJustification(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.False(t, has)
	})
	t.Run("has_justification", func(t *testing.T) {
		client, _, adapter := setupTest(t)

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

func TestCompareAndSetBlockData(t *testing.T) {
	setup := func() (
		*ClientAdapter[Hash, Hasher, Number, Extrinsic, Header],
		offchainapi.OffchainStorage,
	) {
		client, _, adapter := setupTest(t)

		kvdb := memorykvdb.New(13)
		db := database.NewDBAdapter[hash.H256](kvdb)
		storage := offchain.NewLocalStorage(db)

		client.EXPECT().OffchainStorage().Return(storage)

		return adapter, storage
	}

	r := []byte("test_receipt")
	m := []byte("test_message_queue")

	t.Run("no_receipt_and_message_queue", func(t *testing.T) {
		adapter, storage := setup()

		blockHash := common.NewHash([]byte{0})

		bd := &types.BlockData{
			Hash:          blockHash,
			Header:        nil,
			Body:          nil,
			Receipt:       &r,
			MessageQueue:  &m,
			Justification: nil,
		}

		require.Nil(t, storage.Get(state.ReceiptPrefix, blockHash[:]))
		require.Nil(t, storage.Get(state.MessageQueuePrefix, blockHash[:]))

		err := adapter.CompareAndSetBlockData(bd)
		require.NoError(t, err)

		require.Equal(t, r, storage.Get(state.ReceiptPrefix, blockHash[:]))
		require.Equal(t, m, storage.Get(state.MessageQueuePrefix, blockHash[:]))
	})

	t.Run("no_receipt_or_message_queue", func(t *testing.T) {
		adapter, storage := setup()

		blockHash := common.NewHash([]byte{0})

		bd := &types.BlockData{
			Hash:          blockHash,
			Header:        nil,
			Body:          nil,
			Receipt:       nil,
			MessageQueue:  nil,
			Justification: nil,
		}

		// to check that existing values are not overwritten with nil
		storage.Set(state.ReceiptPrefix, blockHash[:], r)
		storage.Set(state.MessageQueuePrefix, blockHash[:], m)

		err := adapter.CompareAndSetBlockData(bd)
		require.NoError(t, err)

		require.Equal(t, r, storage.Get(state.ReceiptPrefix, blockHash[:]))
		require.Equal(t, m, storage.Get(state.MessageQueuePrefix, blockHash[:]))
	})
}
