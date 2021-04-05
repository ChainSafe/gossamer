// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/require"
)

func TestDecodeTransactionHandshake(t *testing.T) {
	testHandshake := &transactionHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeTransactionHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestDecodeTransactionMessage(t *testing.T) {
	testTxMsg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	enc, err := testTxMsg.Encode()
	require.NoError(t, err)

	msg, err := decodeTransactionMessage(enc)
	require.NoError(t, err)
	require.Equal(t, testTxMsg, msg)
}

type mockTransactionHandler struct {
	txs map[common.Hash]types.Extrinsic
}

func newMockTransactionHandler() *mockTransactionHandler {
	return &mockTransactionHandler{
		txs: make(map[common.Hash]types.Extrinsic),
	}
}

func (h *mockTransactionHandler) HandleTransactionMessage(msg *TransactionMessage) error {
	for _, tx := range msg.Extrinsics {
		h.txs[tx.Hash()] = tx
	}

	return nil
}

func TestHandleTransactionMessage(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")

	handler := newMockTransactionHandler()

	config := &Config{
		BasePath:           basePath,
		Port:               7001,
		RandSeed:           1,
		NoBootstrap:        true,
		NoMDNS:             true,
		TransactionHandler: handler,
	}

	s := createTestService(t, config)

	msg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	s.handleTransactionMessage(peer.ID(""), msg)
	for _, tx := range msg.Extrinsics {
		require.NotNil(t, handler.txs[tx.Hash()])
	}
}
