// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeWarpSyncMessage(t *testing.T) {
	t.Parallel()
	testWarpReqMessage := &messages.WarpProofRequest{
		Begin: common.EmptyHash,
	}

	testPeer := peer.ID("me")
	reqEnc, err := testWarpReqMessage.Encode()
	require.NoError(t, err)

	msg, err := decodeWarpSyncMessage(reqEnc, testPeer, true)
	require.NoError(t, err)

	req, ok := msg.(*messages.WarpProofRequest)
	require.True(t, ok)
	require.Equal(t, testWarpReqMessage, req)
}
