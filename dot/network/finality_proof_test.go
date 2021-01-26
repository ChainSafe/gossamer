package network

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestHandleDecodeFinalityProofMessage(t *testing.T) {
	var (
		s = &Service{
			finalityProofRequest: make(map[peer.ID]struct{}),
		}
		req = &FinalityProofRequest{
			Request: []byte("foo"),
		}
		resp = new(FinalityProofResponse)
		peer = peer.ID("anthdm")
	)

	reqB, err := req.Encode()
	require.NoError(t, err)

	msg, err := s.decodeFinalityProofMessage(reqB, peer)
	require.NoError(t, err)

	_, ok := msg.(*FinalityProofRequest)
	require.True(t, ok)

	s.finalityProofRequest[peer] = struct{}{}

	respB, err := resp.Encode()
	require.NoError(t, err)

	msg, err = s.decodeFinalityProofMessage(respB, peer)
	require.NoError(t, err)

	_, ok = msg.(*FinalityProofResponse)
	require.True(t, ok)
}
