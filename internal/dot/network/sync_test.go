// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeSyncMessage(t *testing.T) {
	testPeer := peer.ID("noot")
	reqEnc, err := testBlockRequestMessage.Encode()
	require.NoError(t, err)

	msg, err := decodeSyncMessage(reqEnc, testPeer, true)
	require.NoError(t, err)

	req, ok := msg.(*BlockRequestMessage)
	require.True(t, ok)
	require.Equal(t, testBlockRequestMessage, req)
}
