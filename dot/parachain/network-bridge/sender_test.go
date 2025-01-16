package networkbridge

import (
	"errors"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
)

func TestSendRequests(t *testing.T) {
	t.Run("request_succeeds", func(t *testing.T) {
		request := makeOutgoingRequest(t)
		response := &networkbridgemessages.ChunkFetchingResponse{}
		expectedValue := networkbridgemessages.NoSuchChunk{}
		require.NoError(t, response.SetValue(expectedValue))

		nbs := setUpNetworkBridgeSender(t, response, nil, nil)

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
		requireClosed(t, request.Result)
	})

	t.Run("request_fails", func(t *testing.T) {
		reqErr := errors.New("timeout")
		request := makeOutgoingRequest(t)

		nbs := setUpNetworkBridgeSender(t, nil, nil, reqErr)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.Equal(t, reqErr, result.Error)
		require.Nil(t, result.Response)
		requireClosed(t, request.Result)
	})

	t.Run("decoding_fails", func(t *testing.T) {
		request := makeOutgoingRequest(t)
		rawResponse := []byte("an invalid network response message")

		nbs := setUpNetworkBridgeSender(t, nil, rawResponse, nil)

		sendRequests := networkbridgemessages.SendRequests{
			Requests:       []*networkbridgemessages.OutgoingRequest{request},
			IfDisconnected: networkbridgemessages.TryConnect,
		}

		err := nbs.processMessage(sendRequests)
		require.NoError(t, err)

		result := <-request.Result
		require.Error(t, result.Error)
		require.Nil(t, result.Response)
		requireClosed(t, request.Result)
	})
}

// We arbitrarily use a ChunkFetchingRequest since it does not matter for testing SendRequests handling.
func makeOutgoingRequest(t *testing.T) *networkbridgemessages.OutgoingRequest {
	t.Helper()

	return networkbridgemessages.NewOutgoingRequest(
		"recipient",
		networkbridgemessages.ChunkFetchingRequest{
			CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{1}},
			Index:         42,
		})
}

// only one of response, rawResponse or reqErr should be non-nil
func setUpNetworkBridgeSender(
	t *testing.T,
	response network.ResponseMessage,
	rawResponse []byte,
	reqErr error,
) *NetworkBridgeSender {
	t.Helper()

	if response != nil {
		var err error
		rawResponse, err = response.Encode()
		require.NoError(t, err)
	}

	netService := &mockNetworkService{
		rrp: &mockRequestResponseProtocol{
			rawResponse: rawResponse,
			err:         reqErr,
		},
	}

	return RegisterSender(nil, netService)
}

func requireClosed(t *testing.T, ch chan networkbridgemessages.ReqRespResult) {
	select {
	case <-ch:
	default:
		t.Error("channel is not closed")
	}
}

// TODO use gomock
type mockRequestResponseProtocol struct {
	rawResponse []byte
	err         error
}

func (m *mockRequestResponseProtocol) Do(to peer.ID, req network.Message, response network.ResponseMessage) error {
	if m.err != nil {
		return m.err
	}

	if err := response.Decode(m.rawResponse); err != nil {
		return err
	}
	return nil
}

// TODO use gomock
type mockNetworkService struct {
	rrp *mockRequestResponseProtocol
}

func (m *mockNetworkService) GossipMessage(msg network.NotificationsMessage) {}

func (m *mockNetworkService) SendMessage(to peer.ID, msg network.NotificationsMessage) error {
	return nil
}

func (m *mockNetworkService) RegisterNotificationsProtocol(sub protocol.ID,
	messageID network.MessageType,
	handshakeGetter network.HandshakeGetter,
	handshakeDecoder network.HandshakeDecoder,
	handshakeValidator network.HandshakeValidator,
	messageDecoder network.MessageDecoder,
	messageHandler network.NotificationsMessageHandler,
	batchHandler network.NotificationsMessageBatchHandler,
	maxSize uint64,
) error {
	return nil
}

func (m *mockNetworkService) GetRequestResponseProtocol(
	subprotocol string,
	requestTimeout time.Duration,
	maxResponseSize uint64,
) network.RequestMaker {
	return m.rrp
}

func (m *mockNetworkService) ReportPeer(change peerset.ReputationChange, p peer.ID) {}

func (m *mockNetworkService) DisconnectPeer(setID int, p peer.ID) {}

func (m *mockNetworkService) GetNetworkEventsChannel() chan *network.NetworkEventInfo {
	return nil
}

func (m *mockNetworkService) FreeNetworkEventsChannel(ch chan *network.NetworkEventInfo) {}
