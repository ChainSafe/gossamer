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
