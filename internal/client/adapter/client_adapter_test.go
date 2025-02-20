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
	"github.com/stretchr/testify/require"
)

type Hash = hash.H256
type Hasher = runtime.BlakeTwo256
type Number = uint64
type Extrinsic = runtime.OpaqueExtrinsic
type Header = *generic.Header[Number, Hash, Hasher]

var currentHasher = Hasher{}

func TestBlockStateImplemented(t *testing.T) {
	var _ state.BlockState = &ClientAdapter[Hash, Hasher, Number, Extrinsic, Header]{}
}

func setupTest(t *testing.T) (*mocks.Client[Hash, Hasher, Number, Extrinsic, Header], *mocks.ClientAdapterDB, *ClientAdapter[Hash, Hasher, Number, Extrinsic, Header]) {
	client := mocks.NewClient[Hash, Hasher, Number, Extrinsic, Header](t)
	db := mocks.NewClientAdapterDB(t)
	adapter := NewClientAdapter(client, db)

	return client, db, adapter
}

func TestBestBlock(t *testing.T) {
	bestBlockHash := currentHasher.Hash([]byte("best_block"))
	blockchainInfo := blockchain.Info[Hash, Number]{
		BestHash: bestBlockHash,
	}

	t.Run("best_hash_ok_but_return_error", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		expectedError := errors.New("error")

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Block(bestBlockHash).Return(nil, expectedError)

		block, err := adapter.BestBlock()
		require.Error(t, err)
		require.Nil(t, block)
	})

	t.Run("nil_signed_block", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Block(bestBlockHash).Return(nil, nil)

		block, err := adapter.BestBlock()
		require.NoError(t, err)
		require.Nil(t, block)
	})

	t.Run("ok", func(t *testing.T) {
		client, _, adapter := setupTest(t)

		signedBlock := generic.NewSignedBlock(
			generic.NewBlock[Hasher](
				generic.NewHeader[Number, Hash, Hasher](
					1,
					hash.NewRandomH256(),
					hash.NewRandomH256(),
					hash.NewRandomH256(),
					runtime.Digest{},
				),
				[]Extrinsic{},
			),
			runtime.Justifications{},
		)
		expectedBlock, err := types.FromGenericBlock(signedBlock.Block)
		require.NoError(t, err)

		client.EXPECT().Info().Return(blockchainInfo)
		client.EXPECT().Block(bestBlockHash).Return(signedBlock, nil)

		block, err := adapter.BestBlock()
		require.NoError(t, err)
		require.NotNil(t, block)

		require.Equal(t, expectedBlock, block)
	})
}
