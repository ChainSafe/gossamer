//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeTransactionHandshake(t *testing.T) {
	t.Parallel()

	testHandshake := &transactionHandshake{}

	enc, err := testHandshake.Encode()
	require.NoError(t, err)

	msg, err := decodeTransactionHandshake(enc)
	require.NoError(t, err)
	require.Equal(t, testHandshake, msg)
}

func TestHandleTransactionMessage(t *testing.T) {
	t.Parallel()

	expectedMsgArg := &TransactionMessage{
		Extrinsics: []types.Extrinsic{{1, 1}, {2, 2}},
	}

	ctrl := gomock.NewController(t)
	transactionHandler := NewMockTransactionHandler(ctrl)
	transactionHandler.EXPECT().
		HandleTransactionMessage(peer.ID(""), expectedMsgArg).
		Return(true, nil)

	transactionHandler.EXPECT().
		TransactionsCount().
		Return(0).AnyTimes()

	config := &Config{
		BasePath:           t.TempDir(),
		Port:               availablePort(t),
		NoBootstrap:        true,
		NoMDNS:             true,
		TransactionHandler: transactionHandler,
		telemetryInterval:  time.Hour,
	}

	s := createTestService(t, config)
	time.Sleep(100 * time.Millisecond)
	ret, err := s.handleTransactionMessage(peer.ID(""), expectedMsgArg)

	require.NoError(t, err)
	require.True(t, ret)
}
