// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestDecodeWarpSyncMessage(t *testing.T) {
	t.Parallel()

	// Basic WarpProofRequest
	testWarpReqMessage := &messages.WarpProofRequest{
		Begin: common.MustBlake2bHash([]byte("test")),
	}

	// Test encoding
	reqEnc, err := testWarpReqMessage.Encode()
	require.NoError(t, err)

	//Expected encoded message compared with substrate impl
	require.Equal(t, []byte{
		0x92, 0x8b, 0x20, 0x36, 0x69, 0x43, 0xe2, 0xaf, 0xd1, 0x1e, 0xbc,
		0xe, 0xae, 0x2e, 0x53, 0xa9, 0x3b, 0xf1, 0x77, 0xa4, 0xfc, 0xf3, 0x5b,
		0xcc, 0x64, 0xd5, 0x3, 0x70, 0x4e, 0x65, 0xe2, 0x2,
	}, reqEnc)

	// Test decoding
	testPeer := peer.ID("me")
	msg, err := decodeWarpSyncMessage(reqEnc, testPeer, true)
	require.NoError(t, err)

	req, ok := msg.(*messages.WarpProofRequest)
	require.True(t, ok)
	require.Equal(t, testWarpReqMessage, req)
}

// createServiceWithWarpSyncHelper creates a basic service with warp sync handler support
func createServiceWithWarpSyncHelper(t *testing.T, warpSyncProvider WarpSyncProvider) *Service {
	t.Helper()

	config := &Config{
		BasePath:         t.TempDir(),
		Port:             availablePort(t),
		NoBootstrap:      true,
		NoMDNS:           true,
		WarpSyncProvider: warpSyncProvider,
	}

	srvc := createTestService(t, config)
	srvc.noGossip = true
	handler := newTestStreamHandler(decodeSyncMessage)
	srvc.host.registerStreamHandler(srvc.host.protocolID, handler.handleStream)

	return srvc
}

func TestHandleWarpSyncRequestOk(t *testing.T) {
	t.Parallel()

	// Creates warp sync provider mock to generate proofs with the expected result
	expectedProof := []byte{0x01}

	ctrl := gomock.NewController(t)
	warpSyncProvider := NewMockWarpSyncProvider(ctrl)
	warpSyncProvider.EXPECT().generate(common.EmptyHash).Return(expectedProof, nil).Times(1)

	// Initiate service using the warp sync provider mock
	srvc := createServiceWithWarpSyncHelper(t, warpSyncProvider)

	// Handle request and check resulting proof
	req := messages.WarpProofRequest{
		Begin: common.EmptyHash,
	}

	resp, err := srvc.handleWarpSyncRequest(req)
	require.NoError(t, err)
	require.Equal(t, expectedProof, resp)
}

func TestHandleWarpSyncRequestError(t *testing.T) {
	t.Parallel()

	// Creates warp sync provider mock to generate proofs with the expected erro
	expectedError := fmt.Errorf("error generating proof")
	ctrl := gomock.NewController(t)

	warpSyncProvider := NewMockWarpSyncProvider(ctrl)
	warpSyncProvider.EXPECT().generate(common.EmptyHash).Return(nil, expectedError).Times(1)

	// Initiate service using the warp sync provider mock
	srvc := createServiceWithWarpSyncHelper(t, warpSyncProvider)

	// Handle request and check resulting error
	req := messages.WarpProofRequest{
		Begin: common.EmptyHash,
	}

	resp, err := srvc.handleWarpSyncRequest(req)
	require.Nil(t, resp)
	require.ErrorIs(t, err, expectedError)
}
