package network

import (
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type WarpSyncProvider interface {
	// Generate proof starting at given block hash. The proof is accumulated until maximum proof
	// size is reached.
	generate(start common.Hash) (encodedProof []byte, err error)
}

type WarpSyncRequestHandler struct {
	backend WarpSyncProvider
}

func (w *WarpSyncRequestHandler) handleRequest(req messages.WarpProofRequest) ([]byte, error) {
	// use the backend to generate the warp proof
	proof, err := w.backend.generate(req.Begin)
	if err != nil {
		return nil, err
	}
	// send the response through pendingResponse channel
	return proof, nil
}

func (s *Service) handleWarpSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	s.readStream(stream, decodeSyncMessage, s.handleWarpSyncMessage, MaxBlockResponseSize)
}

func decodeWarpSyncMessage(in []byte, _ peer.ID, _ bool) (messages.P2PMessage, error) {
	msg := new(messages.WarpProofRequest)
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) handleWarpSyncMessage(stream libp2pnetwork.Stream, msg messages.P2PMessage) error {
	if msg == nil {
		return nil
	}

	defer func() {
		err := stream.Close()
		if err != nil && err.Error() != ErrStreamReset.Error() {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	if req, ok := msg.(*messages.WarpProofRequest); ok {
		resp, err := s.warpSyncHandler.handleRequest(*req)
		if err != nil {
			logger.Debugf("cannot create response for request: %s", err)
			return nil
		}

		if _, err = stream.Write(resp); err != nil {
			logger.Debugf("failed to send WarpSyncResponse message to peer %s: %s", stream.Conn().RemotePeer(), err)
			return err
		}
	}

	return nil
}
