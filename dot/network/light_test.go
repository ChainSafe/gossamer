// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestEncodeLightRequest(t *testing.T) {
	t.Parallel()
	exp := common.MustHexToBytes("0x0000000000000000000000000000")

	testLightRequest := NewLightRequest()
	enc, err := testLightRequest.Encode()
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	testLightRequest2 := NewLightRequest()
	err = testLightRequest2.Decode(enc)
	require.NoError(t, err)
	require.Equal(t, testLightRequest, testLightRequest2)
}

func TestEncodeLightResponse(t *testing.T) {
	t.Parallel()
	exp := common.MustHexToBytes("0x00000000000000")

	testLightResponse := NewLightResponse()
	enc, err := testLightResponse.Encode()
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	testLightResponse2 := NewLightResponse()
	for i := range testLightResponse.RemoteHeaderResponse.Header {
		testLightResponse.RemoteHeaderResponse.Header[i] = types.NewEmptyHeader()
	}
	err = testLightResponse2.Decode(enc)
	require.NoError(t, err)
	require.Equal(t, testLightResponse, testLightResponse2)
}

func TestDecodeLightMessage(t *testing.T) {
	s := &Service{
		lightRequest: make(map[peer.ID]struct{}),
	}

	testPeer := peer.ID("noot")

	testLightRequest := NewLightRequest()
	testLightResponse := NewLightResponse()

	reqEnc, err := testLightRequest.Encode()
	require.NoError(t, err)

	msg, err := s.decodeLightMessage(reqEnc, testPeer, true)
	require.NoError(t, err)

	req, ok := msg.(*LightRequest)
	require.True(t, ok)
	resEnc, err := req.Encode()
	require.NoError(t, err)
	require.Equal(t, reqEnc, resEnc)

	s.lightRequest[testPeer] = struct{}{}

	respEnc, err := testLightResponse.Encode()
	require.NoError(t, err)

	msg, err = s.decodeLightMessage(respEnc, testPeer, true)
	require.NoError(t, err)
	resp, ok := msg.(*LightResponse)
	require.True(t, ok)
	resEnc, err = resp.Encode()
	require.NoError(t, err)
	require.Equal(t, respEnc, resEnc)
}

func TestHandleLightMessage_Response(t *testing.T) {
	config := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeA"),
		Port:        7001,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	s := createTestService(t, config)

	configB := &Config{
		BasePath:    utils.NewTestBasePath(t, "nodeB"),
		Port:        7002,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	b := createTestService(t, configB)

	addrInfoB := b.host.addrInfo()
	err := s.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = s.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	stream, err := s.host.h.NewStream(s.ctx, b.host.id(), s.host.protocolID+lightID)
	require.NoError(t, err)

	// Testing empty request
	msg := &LightRequest{}
	err = s.handleLightMsg(stream, msg)
	require.NoError(t, err)

	expectedErr := "failed to find any peer in table"

	// Testing remoteCallResp()
	msg = &LightRequest{
		RemoteCallRequest: &RemoteCallRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteHeaderResp()
	msg = &LightRequest{
		RemoteHeaderRequest: &RemoteHeaderRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteChangeResp()
	msg = &LightRequest{
		RemoteChangesRequest: &RemoteChangesRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadResp()
	msg = &LightRequest{
		RemoteReadRequest: &RemoteReadRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadChildResp()
	msg = &LightRequest{
		RemoteReadChildRequest: &RemoteReadChildRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())
}
