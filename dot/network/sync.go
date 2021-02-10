package network

import (
	"errors"
	"math/rand"
	"sort"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"

	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// handleSyncStream handles streams with the <protocol-id>/sync/2 protocol ID
func (s *Service) handleSyncStream(stream libp2pnetwork.Stream) {
	if stream == nil {
		return
	}

	conn := stream.Conn()
	if conn == nil {
		logger.Error("Failed to get connection from stream")
		return
	}

	peer := conn.RemotePeer()
	s.readStream(stream, peer, s.decodeSyncMessage, s.handleSyncMessage)
}

func (s *Service) decodeSyncMessage(in []byte, peer peer.ID) (Message, error) {
	// s.syncingMu.RLock()
	// defer s.syncingMu.RUnlock()

	// check if we are the requester
	if requested := s.syncQueue.isSyncing(peer); requested {
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
		//s.syncingMu.RLock()
		//if _, isSyncing := s.syncing[peer]; !isSyncing {
		if isSyncing := s.syncQueue.isSyncing(peer); !isSyncing {
			logger.Debug("not currently syncing with peer", "peer", peer)
		}
		//	s.syncingMu.RUnlock()
		return nil
		//}
		//s.syncingMu.RUnlock()

		// req := s.syncer.HandleBlockResponse(resp)
		// if req != nil {
		// 	if err := s.host.send(peer, syncID, req); err != nil {
		// 		s.unsetSyncingPeer(peer)
		// 		logger.Debug("failed to send BlockRequest message; trying other peers", "peer", peer, "error", err)
		// 		s.attemptSyncWithRandomPeer(req)
		// 	}
		// } else {
		// 	// we are done syncing
		// 	s.unsetSyncingPeer(peer)
		// }

		s.syncQueue.pushBlockResponse(resp)
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

func (s *Service) beginSyncing(peer peer.ID, reqs []*BlockRequestMessage) error {
	if len(reqs) == 0 {
		return nil
	}

	// s.syncingMu.Lock()
	// defer s.syncingMu.Unlock()
	// if err := s.setSyncingPeer(peer); err != nil {
	// 	return err
	// }

	// logger.Trace("beginning sync with peer", "peer", peer)

	// err := s.host.send(peer, syncID, msg)
	// if err != nil {
	// 	return err
	// }

	// go s.handleSyncStream(s.host.getStream(peer, syncID))

	//return s.beginSyncingWithPeer(peer, reqs[0])
	s.syncQueue = newSyncQueue(s, peer, reqs)
	s.syncQueue.start()
	return nil

}

type syncQueue struct {
	s *Service
	// TODO: look into scoring for syncing
	peers map[peer.ID]int // peers we have successfully synced from before -> their score; score increases on successful response; decreases otherwise.

	// TODO: move syncing peer map here
	syncing   map[peer.ID]struct{} // set if we have sent a block request message to the given peer
	syncingMu sync.RWMutex

	//requestStarts []uint64 // ordered slice; start of slice is smaller block numbers
	requests  []*BlockRequestMessage // start block of message -> full message
	requestCh chan *BlockRequestMessage

	responseHashes []common.Hash                         // ordered slice; start of slice is earlier blocks, end of slice is later blocks
	responses      map[common.Hash]*BlockResponseMessage // first BlockData hash in response -> full message
	responseCh     chan *BlockRequestMessage

	syncGoal uint64 // current sync goal; ie. what block number we are trying to sync to
}

func newSyncQueue(s *Service, p peer.ID, reqs []*BlockRequestMessage) *syncQueue {
	q := &syncQueue{
		s:              s,
		peers:          map[peer.ID]int{p: 0},
		syncing:        make(map[peer.ID]struct{}),
		requests:       reqs,
		responseHashes: []common.Hash{},
		responses:      make(map[common.Hash]*BlockResponseMessage),
	}

	sortRequests(q.requests)
	return q
}

func sortRequests(reqs []*BlockRequestMessage) {
	sort.Slice(reqs, func(i, j int) bool {
		if reqs[i].StartingBlock.Uint64() < reqs[j].StartingBlock.Uint64() {
			return true
		}

		return false
	})
}

func (q *syncQueue) start() {
	peers := q.getSortedPeers()
	q.beginSyncingWithPeer(peers[0].pid, q.requests[0])

	go func() {
		for {
			//
		}
	}()
}

func (q *syncQueue) stop() {

}

type syncPeer struct {
	pid   peer.ID
	score int
}

func (q *syncQueue) getSortedPeers() []*syncPeer {
	peers := make([]*syncPeer, len(q.peers))
	i := 0
	for pid, score := range q.peers {
		peers[i] = &syncPeer{
			pid:   pid,
			score: score,
		}
		i++
	}

	sort.Slice(peers, func(i, j int) bool {
		if peers[i].score < peers[j].score {
			return true
		}

		return false
	})

	return peers
}

func (q *syncQueue) pushBlockRequest(req *BlockRequestMessage) {
	q.requests = append(q.requests, req)
	sortRequests(q.requests)
}

func (q *syncQueue) pushBlockResponse(resp *BlockResponseMessage) {
	if _, has := q.responses[resp.BlockData[0].Hash]; has {
		return
	}

	//q.responseHashes =
}

func (q *syncQueue) processBlockRequest() {
	for req := range q.requestCh {

	}
}

func (q *syncQueue) processBlockResponses() {
	for resp := range q.responseCh {

	}
}

func (q *syncQueue) attemptSyncWithRandomPeer(req *BlockRequestMessage) {
	peers := q.s.host.peers()
	rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

	for _, peer := range peers {
		// s.syncingMu.Lock()
		// if err := s.host.send(peer, syncID, req); err == nil {
		// 	go s.handleSyncStream(s.host.getStream(peer, syncID))
		// 	_ = s.setSyncingPeer(peer)
		// 	s.syncingMu.Unlock()
		// 	break
		// }
		// s.syncingMu.Unlock()
		if err := q.beginSyncingWithPeer(peer, req); err == nil {
			logger.Debug("successfully began sync with peer", "peer", peer)
			return
		}
	}

	logger.Warn("failed to begin sync with any peer")
}

func (q *syncQueue) beginSyncingWithPeer(peer peer.ID, req *BlockRequestMessage) error {
	q.syncingMu.Lock()
	defer q.syncingMu.Unlock()

	if _, syncing := q.syncing[peer]; syncing {
		return errors.New("already syncing with peer")
	}

	q.syncing[peer] = struct{}{}
	q.s.host.h.ConnManager().Protect(peer, "")

	logger.Trace("beginning sync with peer", "peer", peer)

	err := q.s.host.send(peer, syncID, req)
	if err != nil {
		return err
	}

	go q.s.handleSyncStream(q.s.host.getStream(peer, syncID))
	return nil
}

func (q *syncQueue) unsetSyncingPeer(peer peer.ID) {
	q.syncingMu.Lock()
	defer q.syncingMu.Unlock()

	delete(q.syncing, peer)
	q.s.host.h.ConnManager().Unprotect(peer, "")
}

func (q *syncQueue) isSyncing(peer peer.ID) bool {
	_, syncing := q.syncing[peer]
	return syncing
}
