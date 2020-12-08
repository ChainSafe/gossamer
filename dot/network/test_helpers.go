package network

import (
	"bufio"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type mockSyncer struct {
	highestSeen *big.Int
	synced      bool
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{
		highestSeen: big.NewInt(0),
		synced:      false,
	}
}

func (s *mockSyncer) CreateBlockResponse(msg *BlockRequestMessage) (*BlockResponseMessage, error) {
	return nil, nil
}

func (s *mockSyncer) HandleBlockResponse(msg *BlockResponseMessage) *BlockRequestMessage {
	return nil
}

func (s *mockSyncer) HandleBlockAnnounce(msg *BlockAnnounceMessage) *BlockRequestMessage {
	if msg.Number.Cmp(s.highestSeen) > 0 {
		s.highestSeen = msg.Number
	}

	startBlock, _ := variadic.NewUint64OrHash(1)
	return &BlockRequestMessage{
		ID:            99,
		StartingBlock: startBlock,
		Max:           optional.NewUint32(false, 0),
	}
}

func (s *mockSyncer) IsSynced() bool {
	return s.synced
}

func (s *mockSyncer) SetSyncedState(newState bool) {
	s.synced = newState
}

type testStreamHandler struct {
	messages map[peer.ID]Message
}

func newTestStreamHandler() *testStreamHandler {
	return &testStreamHandler{
		messages: make(map[peer.ID]Message),
	}
}

func (s *testStreamHandler) handleStream(stream libp2pnetwork.Stream) {
	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, decodeMessageBytes, s.handleMessage)
}

func (s *testStreamHandler) handleMessage(peer peer.ID, msg Message) error {
	s.messages[peer] = msg
	return nil
}

func (s *testStreamHandler) readStream(stream libp2pnetwork.Stream, peer peer.ID, decoder messageDecoder, handler messageHandler) {
	// create buffer stream for non-blocking read
	r := bufio.NewReader(stream)

	for {
		length, err := readLEB128ToUint64(r)
		if err != nil {
			logger.Error("Failed to read LEB128 encoding", "error", err)
			_ = stream.Close()
			return
		}

		if length == 0 {
			continue
		}

		msgBytes := make([]byte, length)
		tot := uint64(0)
		for i := 0; i < maxReads; i++ {
			n, err := r.Read(msgBytes[tot:]) //nolint
			if err != nil {
				logger.Error("Failed to read message from stream", "error", err)
				_ = stream.Close()
				return
			}

			tot += uint64(n)
			if tot == length {
				break
			}
		}

		if tot != length {
			logger.Error("Failed to read entire message", "length", length, "read" /*n*/, tot)
			continue
		}

		// decode message based on message type
		msg, err := decoder(msgBytes, peer)
		if err != nil {
			logger.Error("Failed to decode message from peer", "peer", peer, "err", err)
			continue
		}

		// handle message based on peer status and message type
		err = handler(peer, msg)
		if err != nil {
			logger.Error("Failed to handle message from stream", "message", msg, "error", err)
			_ = stream.Close()
			return
		}
	}
}
