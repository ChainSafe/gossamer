package network

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)
//
//func TestEncoding(t *testing.T) {
//	testLightRequest := NewLightRequest()
//	testLightRequestNew := NewLightRequestNew()
//
//	enc1, err := scaleOld.Encode(testLightRequest.RmtCallRequest)
//	require.NoError(t, err)
//
//	enc2, err := scale.Marshal(testLightRequestNew.RmtCallRequest)
//	require.NoError(t, err)
//	require.Equal(t, enc1, enc2)
//
//	enc1, err = scaleOld.Encode(testLightRequest.RmtReadRequest)
//	require.NoError(t, err)
//
//	enc2, err = scale.Marshal(testLightRequestNew.RmtReadRequest)
//	require.NoError(t, err)
//	require.Equal(t, enc1, enc2)
//
//	enc1, err = scaleOld.Encode(testLightRequest.RmtHeaderRequest)
//	require.NoError(t, err)
//
//	enc2, err = scale.Marshal(testLightRequestNew.RmtHeaderRequest)
//	require.NoError(t, err)
//	require.Equal(t, enc1, enc2)
//
//	enc1, err = scaleOld.Encode(testLightRequest.RmtReadChildRequest)
//	require.NoError(t, err)
//
//	enc2, err = scale.Marshal(testLightRequestNew.RmtReadChildRequest)
//	require.NoError(t, err)
//	require.Equal(t, enc1, enc2)
//
//	enc1, err = scaleOld.Encode(testLightRequest.RmtChangesRequest)
//	require.NoError(t, err)
//
//	enc2, err = scale.Marshal(testLightRequestNew.RmtChangesRequest)
//	require.NoError(t, err)
//	require.Equal(t, enc1, enc2)
//}


func TestDecodeLightMessage(t *testing.T) {
	s := &Service{
		lightRequest: make(map[peer.ID]struct{}),
	}

	testPeer := peer.ID("noot")

	testLightRequest := NewLightRequest()
	testLightResponse := NewLightResponse()

	testLightRequestNew := NewLightRequestNew()

	reqEnc, err := testLightRequest.Encode()
	require.NoError(t, err)

	reqEncNew, err := scale.Marshal(*testLightRequestNew)
	require.NoError(t, err)

	//expReq := common.MustHexToBytes("0x000000000000000000000000")
	require.Equal(t, reqEnc, reqEncNew)

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
		RmtCallRequest: &RemoteCallRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteHeaderResp()
	msg = &LightRequest{
		RmtHeaderRequest: &RemoteHeaderRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteChangeResp()
	msg = &LightRequest{
		RmtChangesRequest: &RemoteChangesRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadResp()
	msg = &LightRequest{
		RmtReadRequest: &RemoteReadRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())

	// Testing remoteReadChildResp()
	msg = &LightRequest{
		RmtReadChildRequest: &RemoteReadChildRequest{},
	}
	err = s.handleLightMsg(stream, msg)
	require.Error(t, err, expectedErr, msg.String())
}
