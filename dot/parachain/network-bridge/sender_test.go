// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"errors"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendRequests(t *testing.T) {
	t.Parallel()

	t.Run("request_succeeds", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		request := makeOutgoingRequest(t)
		response := &networkbridgemessages.ChunkFetchingResponse{}
		expectedValue := networkbridgemessages.NoSuchChunk{}
		require.NoError(t, response.SetValue(expectedValue))

		nbs := setUpNetworkBridgeSender(t, ctrl, request, response, nil, nil)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.NoError(t, result.Error)

		cfResponse, ok := result.Response.(*networkbridgemessages.ChunkFetchingResponse)
		require.True(t, ok)

		actualValue, err := cfResponse.Value()
		require.NoError(t, err)

		require.Equal(t, expectedValue, actualValue)
		requireEmptyAndClosed(t, request.Result)
	})

	t.Run("request_fails", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		reqErr := errors.New("timeout")
		request := makeOutgoingRequest(t)

		nbs := setUpNetworkBridgeSender(t, ctrl, request, nil, nil, reqErr)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.Equal(t, reqErr, result.Error)
		require.Nil(t, result.Response)
		requireEmptyAndClosed(t, request.Result)
	})

	t.Run("decoding_fails", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		request := makeOutgoingRequest(t)
		rawResponse := []byte("an invalid network response message")

		nbs := setUpNetworkBridgeSender(t, ctrl, request, nil, rawResponse, nil)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.Error(t, result.Error)
		require.Nil(t, result.Response)
		requireEmptyAndClosed(t, request.Result)
	})

	t.Run("cancel_request", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		request := makeOutgoingRequest(t)
		nbs := setUpNetworkBridgeSender(t, ctrl, request, nil, nil, nil)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		request.Cancel()

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.Nil(t, result.Response)
		require.NoError(t, result.Error)
		requireEmptyAndClosed(t, request.Result)
	})
}

// We arbitrarily use a ChunkFetchingRequest since it does not matter for testing SendRequests handling.
func makeOutgoingRequest(t *testing.T) *networkbridgemessages.OutgoingRequest {
	t.Helper()

	return networkbridgemessages.NewOutgoingRequest(
		parachaintypes.PeerID("recipient"),
		&networkbridgemessages.ChunkFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{1}},
			Index:         42,
		})
}

// Expect calls to Network.GetRequestResponseProtocol() and RequestMaker.Do() when only one of response, rawResponse or
// reqErr is non-nil.
// Expect no calls Network.GetRequestResponseProtocol() RequestMaker.Do() when all three are nil.
func setUpNetworkBridgeSender(
	t *testing.T,
	ctrl *gomock.Controller,
	request *networkbridgemessages.OutgoingRequest,
	response network.ResponseMessage,
	rawResponse []byte,
	reqErr error,
) *NetworkBridgeSender {
	t.Helper()

	expectCancellation := response == nil && rawResponse == nil && reqErr == nil

	paraPeerID, ok := request.Recipient.(parachaintypes.PeerID)
	require.True(t, ok)
	expectedPeer := peer.ID(paraPeerID)

	reqMaker := NewMockRequestMaker(ctrl)

	if expectCancellation {
		reqMaker.EXPECT().
			Do(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)
	} else {
		reqMaker.EXPECT().
			Do(expectedPeer, request.Payload, gomock.AssignableToTypeOf(request.Payload.Response())).
			DoAndReturn(func(to peer.ID, req network.Message, res network.ResponseMessage) error {
				if reqErr != nil {
					return reqErr
				}

				if response != nil {
					var err error
					rawResponse, err = response.Encode()
					require.NoError(t, err)
				}

				if err := res.Decode(rawResponse); err != nil {
					return err
				}
				return nil
			})
	}

	netService := NewMockNetwork(ctrl)

	if expectCancellation {
		netService.EXPECT().
			GetRequestResponseProtocol(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)
	} else {
		netService.EXPECT().
			GetRequestResponseProtocol(request.Payload.Protocol().String(), gomock.Any(), gomock.Any()).
			Return(reqMaker)
	}

	return RegisterSender(nil, netService)
}

func requireEmptyAndClosed(t *testing.T, ch chan networkbridgemessages.ReqRespResult) {
	select {
	case _, ok := <-ch:
		require.False(t, ok, "channel was not empty")
	default:
		t.Error("channel is not closed")
	}
}
