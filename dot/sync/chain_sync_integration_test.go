//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/peer"
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
		RequestedData: bootstrapRequestData,
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

func TestChainSync_validateResponse_firstBlock_Integration(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := newTestChainSync(ctrl)
	bs := NewMockBlockState(ctrl)
	bs.EXPECT().HasHeader(gomock.AssignableToTypeOf(common.Hash{})).Return(false, nil)
	cs.blockState = bs

	req := &network.BlockRequestMessage{
		RequestedData: bootstrapRequestData,
	}

	header := &types.Header{
		Number: 2,
	}

	resp := &network.BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Hash: header.Hash(),
				Header: &types.Header{
					Number: 2,
				},
				Body:          &types.Body{},
				Justification: &[]byte{0},
			},
		},
	}

	err := cs.validateResponse(req, resp, "")
	require.True(t, errors.Is(err, errUnknownParent))
	require.True(t, cs.pendingBlocks.(*disjointBlockSet).hasBlock(header.Hash()))
	bd := cs.pendingBlocks.getBlock(header.Hash())
	require.NotNil(t, bd.header)
	require.NotNil(t, bd.body)
	require.NotNil(t, bd.justification)
}
