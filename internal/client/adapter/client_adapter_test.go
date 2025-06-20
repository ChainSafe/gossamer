// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/client/adapter/mocks"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
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
			expectedHeader, err := types.NewHeaderFromGeneric(header)
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

func TestGetJustification(t *testing.T) {
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

		expectedBlock, err := types.NewBlockFromGeneric[Number, Hash, Extrinsic](block)
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
	t.Parallel()

	t.Run("header_is_not_finalized", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Children(blockchainInfo.FinalizedHash).Return([]Hash{}, nil)
		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		has, err := adapter.HasHeader(common.NewHashFromGeneric(blockHash))
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("not_has_header", func(t *testing.T) {
		t.Parallel()

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

func TestGetStateRootFromBlock(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")
		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		bhash := common.NewHashFromGeneric(blockchainInfo.FinalizedHash)

		root, err := adapter.GetStateRootFromBlock(&bhash)
		require.Error(t, err, expectedError)
		require.Nil(t, root)
	})

	t.Run("block_not_found", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(nil, nil)

		bhash := common.NewHashFromGeneric(blockchainInfo.FinalizedHash)

		root, err := adapter.GetStateRootFromBlock(&bhash)
		require.ErrorIs(t, err, database.ErrNotFound)
		require.Nil(t, root)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Header(blockchainInfo.FinalizedHash).Return(&header, nil)

		expectedRoot := common.NewHashFromGeneric(header.StateRoot())
		bhash := common.NewHashFromGeneric(blockchainInfo.FinalizedHash)

		root, err := adapter.GetStateRootFromBlock(&bhash)
		require.NoError(t, err)
		require.Equal(t, &expectedRoot, root)
	})

	t.Run("bhash_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(blockchainInfo.BestHash).Return(&header, nil)

		expectedRoot := common.NewHashFromGeneric(header.StateRoot())

		root, err := adapter.GetStateRootFromBlock(nil)

		require.NoError(t, err)
		require.Equal(t, &expectedRoot, root)
	})
}

func TestGetStorage(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("kaput")
		client.EXPECT().StateAt(header.Hash()).Return(nil, expectedError)
		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(header.Hash()).Return(&header, nil)

		root := common.NewHashFromGeneric(header.StateRoot())

		value, err := adapter.GetStorage(&root, []byte("key"))
		require.ErrorIs(t, err, expectedError)
		require.Nil(t, value)
	})
	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)
		backend.EXPECT().Storage([]byte("key")).Return([]byte("value"), nil)

		client.EXPECT().StateAt(header.Hash()).Return(backend, nil)
		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Header(header.Hash()).Return(&header, nil)

		root := common.NewHashFromGeneric(header.StateRoot())

		value, err := adapter.GetStorage(&root, []byte("key"))
		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})
	t.Run("root_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)
		backend.EXPECT().Storage([]byte("key")).Return([]byte("value"), nil)

		client.EXPECT().StateAt(blockchainInfo.BestHash).Return(backend, nil)
		client.EXPECT().Info().Return(blockchainInfo)

		value, err := adapter.GetStorage(nil, []byte("key"))
		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})
	t.Run("search_succeeds", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)
		backend.EXPECT().Storage([]byte("key")).Return([]byte("value"), nil)

		length := min(maxSearchDepth-1, 10)
		genesisHeader := makeHeaderChain(t, client, uint16(length))
		client.EXPECT().StateAt(genesisHeader.Hash()).Return(backend, nil)

		root := common.NewHashFromGeneric(genesisHeader.StateRoot())

		value, err := adapter.GetStorage(&root, []byte("key"))
		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})
	t.Run("search_fails", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)

		backend.EXPECT().
			Storage([]byte("key")).
			Maybe().
			Return([]byte("value"), nil)

		genesisHeader := makeHeaderChain(t, client, maxSearchDepth+1)
		client.EXPECT().StateAt(genesisHeader.Hash()).Maybe().Return(backend, nil)

		root := common.NewHashFromGeneric(genesisHeader.StateRoot())

		value, err := adapter.GetStorage(&root, []byte("key"))
		require.Error(t, err)
		require.Nil(t, value)
	})
}

// makeHeaderChain creates a chain of headers and returns the header of the genesis block.
//
// The given client is configured to return them from Header() by their hash and to return
// [blockchain.Info] with appropriate values from Info().
func makeHeaderChain(
	t *testing.T,
	client *mocks.Client[Hash, Hasher, Number, Extrinsic, Header],
	length uint16,
) (genesisHeader Header) {
	t.Helper()

	genesisHeader = header.Clone().(Header)
	genesisHeader.SetNumber(Number(0))
	client.EXPECT().Header(genesisHeader.Hash()).Maybe().Return(&genesisHeader, nil)
	chain := make([]Header, length)
	chain[0] = genesisHeader
	hasher := new(Hasher)

	for i := 1; i < int(length); i++ {
		h := generic.NewHeader[Number, Hash, Hasher](
			Number(i),
			hash.NewRandomH256(),
			hasher.Hash([]byte(fmt.Sprintf("header%d", i))),
			chain[i-1].Hash(),
			runtime.Digest{},
		)

		chain[i] = h
		client.EXPECT().Header(h.Hash()).Maybe().Return(&h, nil)
	}

	info := blockchain.Info[Hash, Number]{
		GenesisHash:     genesisHeader.Hash(),
		BestHash:        chain[length-1].Hash(),
		BestNumber:      chain[length-1].Number(),
		FinalizedHash:   chain[length-1].Hash(),
		FinalizedNumber: chain[length-1].Number(),
	}

	client.EXPECT().Info().Return(info)

	return
}

func TestGetStorageByBlockHash(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("kaput")
		client.EXPECT().StateAt(blockchainInfo.FinalizedHash).Return(nil, expectedError)

		bhash := common.NewHashFromGeneric(blockchainInfo.FinalizedHash)

		value, err := adapter.GetStorageByBlockHash(&bhash, []byte("key"))
		require.ErrorIs(t, err, expectedError)
		require.Nil(t, value)
	})
	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)
		backend.EXPECT().Storage([]byte("key")).Return([]byte("value"), nil)

		client.EXPECT().StateAt(blockchainInfo.FinalizedHash).Return(backend, nil)

		bhash := common.NewHashFromGeneric(blockchainInfo.FinalizedHash)

		value, err := adapter.GetStorageByBlockHash(&bhash, []byte("key"))
		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})
	t.Run("bhash_nil", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		backend := mocks.NewStatemachineBackend[Hash, Hasher](t)
		backend.EXPECT().Storage([]byte("key")).Return([]byte("value"), nil)

		client.EXPECT().StateAt(blockchainInfo.BestHash).Return(backend, nil)
		client.EXPECT().Info().Return(blockchainInfo)

		value, err := adapter.GetStorageByBlockHash(nil, []byte("key"))
		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})
}

func Test_GetKeysWithPrefix_Entries(t *testing.T) {
	client, _, adapter := setupTest(t)

	entries := map[string][]byte{
		"":        {},
		"key1":    []byte("value1"),
		"key2":    []byte("value2"),
		"xyzKey1": []byte("xyzValue1"),
		"long":    []byte("newvaluewithmorethan32byteslength"),
	}

	backend := statemachine.NewMemoryDBTrieBackendFromMap[Hash, Hasher](entries, storage.StateVersionV1)
	client.EXPECT().Header(header.Hash()).Return(&header, nil)
	client.EXPECT().StateAt(header.Hash()).Return(backend, nil)
	client.EXPECT().Info().Return(blockchainInfo)

	root := common.NewHashFromGeneric(header.StateRoot())

	t.Run("GetKeysWithPrefix", func(t *testing.T) {
		keys, err := adapter.GetKeysWithPrefix(&root, []byte("ke"))

		require.NoError(t, err)
		require.Len(t, keys, 2)
		require.Contains(t, keys, []byte("key1"))
		require.Contains(t, keys, []byte("key2"))
	})

	t.Run("GetKeysWithPrefix", func(t *testing.T) {
		result, err := adapter.Entries(&root)

		require.NoError(t, err)
		require.Equal(t, entries, result)
	})
}

func TestGetFinalisedHeader(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		db.EXPECT().Get(state.FinalisedHashKey(23, 5)).Return(nil, database.ErrNotFound)

		finalisedHeader, err := adapter.GetFinalisedHeader(23, 5)

		require.ErrorIs(t, err, database.ErrNotFound)
		require.Nil(t, finalisedHeader)
	})
	t.Run("ok", func(t *testing.T) {
		client, db, adapter := setupTest(t)

		client.EXPECT().Block(blockHash).Return(signedBlock, nil)

		expectedHeader, err := types.NewHeaderFromGeneric(header)
		require.NoError(t, err)

		db.EXPECT().Get(state.FinalisedHashKey(23, 5)).Return(blockHash.Bytes(), nil)

		finalisedHeader, err := adapter.GetFinalisedHeader(23, 5)

		require.NoError(t, err)
		require.Equal(t, expectedHeader, finalisedHeader)
	})
}

func TestGetFirstNonOriginSlotNumber(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		db.EXPECT().Get(state.FirstSlotNumberKey).Return(nil, database.ErrNotFound)

		num, err := adapter.GetFirstNonOriginSlotNumber()

		require.NoError(t, err)
		require.Equal(t, uint64(0), num)
	})
	t.Run("other_error", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		db.EXPECT().Get(state.FirstSlotNumberKey).Return(nil, errors.New("kaput"))

		num, err := adapter.GetFirstNonOriginSlotNumber()

		require.Error(t, err)
		require.Equal(t, uint64(0), num)
	})
	t.Run("ok", func(t *testing.T) {
		_, db, adapter := setupTest(t)

		fiveEncoded := []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

		db.EXPECT().Get(state.FirstSlotNumberKey).Return(fiveEncoded[:], nil)

		num, err := adapter.GetFirstNonOriginSlotNumber()

		require.NoError(t, err)
		require.Equal(t, uint64(5), num)
	})
}

func Test_GetStorageFromChild_GetStorageChild(t *testing.T) {
	client, _, adapter := setupTest(t)

	childInfo := storage.NewDefaultChildInfo([]byte("child1"))
	mdb := trie.NewPrefixedMemoryDB[Hash, Hasher]()

	ksdb := trie.NewKeyspacedDB(mdb, childInfo.Keyspace())
	childTrie := triedb.NewEmptyTrieDB[Hash, Hasher](ksdb)
	childTrie.SetVersion(storage.DefaultStateVersion.TrieLayout())
	require.NoError(t, childTrie.Set([]byte("key"), []byte("value")))
	require.NoError(t, childTrie.Set([]byte("anotherkey"), []byte("anothervalue")))

	parentTrie := triedb.NewEmptyTrieDB[Hash, Hasher](mdb)
	require.NoError(t, parentTrie.Set(childInfo.PrefixedStorageKey(), childTrie.MustHash().Bytes()))
	require.NoError(t, parentTrie.Set([]byte("key"), []byte("value2")))
	require.NoError(t, parentTrie.Set([]byte(":code"), []byte("return 42")))
	parentTrie.SetVersion(storage.DefaultStateVersion.TrieLayout())

	header := header.Clone().(Header)
	header.SetStateRoot(parentTrie.MustHash())

	info := blockchainInfo
	info.BestHash = header.Hash()
	client.EXPECT().Info().Return(info)

	backend := statemachine.NewTrieBackend[Hash, Hasher](
		statemachine.HashDBTrieBackendStorage[Hash]{HashDB: mdb},
		parentTrie.MustHash(),
		nil,
		nil,
	)
	client.EXPECT().StateAt(header.Hash()).Return(backend, nil)

	t.Run("GetStorageFromChild", func(t *testing.T) {
		value, err := adapter.GetStorageFromChild(nil, []byte("child1"), []byte("key"))

		require.NoError(t, err)
		require.Equal(t, []byte("value"), value)
	})

	t.Run("GetStorageChild", func(t *testing.T) {
		storageChild, err := adapter.GetStorageChild(nil, []byte("child1"))

		require.NoError(t, err)

		require.Equal(
			t,
			map[string][]byte{
				"key":        []byte("value"),
				"anotherkey": []byte("anothervalue"),
			},
			storageChild.Entries(),
		)
	})
}
