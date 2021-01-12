package network

import (
	"errors"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// handleSyncStream handles streams with the <protocol-id>/sync/2 protocol ID
func (s *Service) handleSyncStream(stream libp2pnetwork.Stream) {
	logger.Info("handling sync stream")
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, s.decodeSyncMessage, s.handleSyncMessage)
}

func (s *Service) decodeSyncMessage(in []byte, peer peer.ID) (Message, error) {
	s.syncingMu.RLock()
	defer s.syncingMu.RUnlock()

	// check if we are the requester
	if _, requested := s.syncing[peer]; requested {
		// if we are, decode the bytes as a BlockResponseMessage
		msg := new(BlockResponseMessage)
		err := msg.Decode(in)
		return msg, err
	}

	// otherwise, decode bytes as BlockRequestMessage
	msg := new(BlockRequestMessage)
	err := msg.Decode(in)
	return msg, err
}

// handleSyncMessage handles synchronization message types (BlockRequest and BlockResponse)
func (s *Service) handleSyncMessage(peer peer.ID, msg Message) error {
	if msg == nil {
		return nil
	}

	if resp, ok := msg.(*BlockResponseMessage); ok {
		if _, isSyncing := s.syncing[peer]; !isSyncing {
			logger.Debug("not currently syncing with peer", "peer", peer)
			return nil
		}

		req := s.syncer.HandleBlockResponse(resp)
		if req != nil {
			s.syncing[peer] = struct{}{}
			err := s.host.send(peer, syncID, req)
			if err != nil {
				logger.Error("failed to send BlockRequest message", "peer", peer)
			}
		} else {
			// we are done syncing
			delete(s.syncing, peer)
			// TODO: close stream
		}
	}

	// if it's a BlockRequest, call core for processing
	if req, ok := msg.(*BlockRequestMessage); ok {
		resp, err := s.syncer.CreateBlockResponse(req)
		if err != nil {
			logger.Debug("cannot create response for request")
			// TODO: close stream
			return nil
		}

		err = s.host.send(peer, syncID, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", peer)
		}
	}

	return nil
}

func (s *Service) beginSyncing(peer peer.ID, msg Message) error {
	if _, syncing := s.syncing[peer]; syncing {
		return errors.New("already syncing with peer")
	}

	logger.Trace("beginning sync with peer", "peer", peer, "msg", msg)
	s.syncing[peer] = struct{}{}
	err := s.host.send(peer, syncID, msg)
	if err != nil {
		return err
	}

	go s.handleSyncStream(s.host.getStream(peer, syncID))
	return nil
}
