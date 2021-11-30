// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/mock"
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

func TestHandleTransactionMessage(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	mockhandler := &MockTransactionHandler{}
	mockhandler.On("HandleTransactionMessage",
		mock.AnythingOfType("peer.ID"),
		mock.AnythingOfType("*network.TransactionMessage")).
		Return(true, nil)
	mockhandler.On("TransactionsCount").Return(0)

	config := &Config{
		BasePath:           basePath,
		Port:               7001,
		NoBootstrap:        true,
		NoMDNS:             true,
		TransactionHandler: mockhandler,
	}

	s := createTestService(t, config)

	msg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	s.handleTransactionMessage(peer.ID(""), msg)
	mockhandler.AssertCalled(t, "HandleTransactionMessage",
		mock.AnythingOfType("peer.ID"), msg)
}
