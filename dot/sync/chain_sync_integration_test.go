//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestValidateBlockData(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	mockNetwork := NewMockNetwork(ctrl)
	mockNetwork.EXPECT().ReportPeer(peerset.ReputationChange{
		Value:  -1048576,
		Reason: "Incomplete header",
	}, peer.ID(""))
	cs.network = mockNetwork

	req := &network.BlockRequestMessage{
		RequestedData: network.BootstrapRequestData,
	}

	err := cs.validateBlockData(req, nil, "")
	require.Equal(t, errNilBlockData, err)

	err = cs.validateBlockData(req, &types.BlockData{}, "")
	require.Equal(t, errNilHeaderInResponse, err)

	err = cs.validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
	}, "")
	require.ErrorIs(t, err, errNilBodyInResponse)

	err = cs.validateBlockData(req, &types.BlockData{
		Header: &types.Header{},
		Body:   &types.Body{},
	}, "")
	require.NoError(t, err)
}
