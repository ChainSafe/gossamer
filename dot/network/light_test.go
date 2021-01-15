package network

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestDecodeLightMessage(t *testing.T) {
	s := &Service{
		lightRequest: make(map[peer.ID]struct{}),
	}

	testPeer := peer.ID("noot")

	testLightRequest := NewLightRequest()
	testLightResponse := NewLightResponse()

	reqEnc, err := testLightRequest.Encode()
	require.NoError(t, err)

	msg, err := s.decodeLightMessage(reqEnc, testPeer)
	require.NoError(t, err)

	req, ok := msg.(*LightRequest)
	require.True(t, ok)
	resEnc, err := req.Encode()
	require.NoError(t, err)
	require.Equal(t, reqEnc, resEnc)

	s.lightRequest[testPeer] = struct{}{}

	respEnc, err := testLightResponse.Encode()
	require.NoError(t, err)

	msg, err = s.decodeLightMessage(respEnc, testPeer)
	require.NoError(t, err)
	resp, ok := msg.(*LightResponse)
	require.True(t, ok)
	resEnc, err = resp.Encode()
	require.NoError(t, err)
	require.Equal(t, respEnc, resEnc)
}

func TestHandleLightMessage_Response(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "nodeA")
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}
	s := createTestService(t, config)

	peerID := peer.ID("noot")

	// Testing empty request
	msg := &LightRequest{}
	err := s.handleLightMsg(peerID, msg)
	require.NoError(t, err)

	expectedErr := "failed to find any peer in table"

	// Testing remoteCallResp()
	msg = &LightRequest{
		RmtCallRequest: &RemoteCallRequest{},
	}
	err = s.handleLightMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteHeaderResp()
	msg = &LightRequest{
		RmtHeaderRequest: &RemoteHeaderRequest{},
	}
	err = s.handleLightMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteChangeResp()
	msg = &LightRequest{
		RmtChangesRequest: &RemoteChangesRequest{},
	}
	err = s.handleLightMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadResp()
	msg = &LightRequest{
		RmtReadRequest: &RemoteReadRequest{},
	}
	err = s.handleLightMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadChildResp()
	msg = &LightRequest{
		RmtReadChildRequest: &RemoteReadChildRequest{},
	}
	err = s.handleLightMsg(peerID, msg)
	require.Error(t, err, expectedErr, msg.String())
}
