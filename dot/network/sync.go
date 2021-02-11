package network

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"time"

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
		if isSyncing := s.syncQueue.isSyncing(peer); !isSyncing {
			logger.Debug("not currently syncing with peer", "peer", peer)
			return nil
		}

		s.syncQueue.pushBlockResponse(resp, peer)
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

type syncPeer struct { //nolint
	pid   peer.ID
	score int
}

type syncRequest struct {
	req *BlockRequestMessage
	pid peer.ID // rename to "to"?
}

type syncResponse struct {
	resp       *BlockResponseMessage
	start, end int64
	pid        peer.ID // rename to "from"?
}

type syncQueue struct {
	s         *Service
	ctx       context.Context
	cancel    context.CancelFunc
	peerScore map[peer.ID]int // peers we have successfully synced from before -> their score; score increases on successful response; decreases otherwise.

	syncing   map[peer.ID]struct{} // set if we have sent a block request message to the given peer
	syncingMu sync.RWMutex

	requests    []*syncRequest // start block of message -> full message
	requestCh   chan *syncRequest
	requestLock sync.RWMutex

	responses    []*syncResponse
	responseCh   chan *syncResponse
	responseLock sync.RWMutex
}

func newSyncQueue(s *Service) *syncQueue {
	ctx, cancel := context.WithCancel(s.ctx)

	return &syncQueue{
		s:          s,
		ctx:        ctx,
		cancel:     cancel,
		peerScore:  make(map[peer.ID]int),
		syncing:    make(map[peer.ID]struct{}),
		requests:   []*syncRequest{},
		requestCh:  make(chan *syncRequest),
		responses:  []*syncResponse{},
		responseCh: make(chan *syncResponse),
	}
}

func (q *syncQueue) start() {
	go func() {
		for {
			select {
			case <-time.After(time.Second):
			case <-q.ctx.Done():
				return
			}

			// if we have block requests to send, put them into requestCh
			if len(q.requests) == 0 {
				continue
			}

			q.requestLock.Lock()
			logger.Debug("sync request queue", "queue", q.stringifyRequestQueue())
			q.requestCh <- q.requests[0]
			q.requests = q.requests[1:]
			q.requestLock.Unlock()
		}
	}()

	go func() {
		for {
			select {
			case <-time.After(time.Second):
			case <-q.ctx.Done():
				return
			}

			head, err := q.s.blockState.BestBlockNumber()
			if err != nil {
				continue
			}

			// if we have block requests to send, put them into requestCh
			q.responseLock.Lock()
			if len(q.responses) == 0 {
				q.responseLock.Unlock()
				continue
			}

			if q.responses[0].end <= head.Int64() {
				// this assumes responses are always in ascending order, which should be true as long as other nodes respect our requested direction
				logger.Debug("response end was less than head+1, removing", "queue end", q.responses[0].end, "head+1", head.Int64()+1)
				q.responses = q.responses[1:]
				q.responseLock.Unlock()
				continue
			}

			logger.Debug("sync response queue", "queue", q.stringifyResponseQueue())
			q.responseCh <- q.responses[0]
			q.responses = q.responses[1:]
			q.responseLock.Unlock()
		}
	}()

	go q.processBlockRequest()
	go q.processBlockResponses()
}

func (q *syncQueue) stringifyRequestQueue() string {
	str := ""
	for _, req := range q.requests {
		str = str + fmt.Sprintf("[start=%d end=%d] ", req.req.StartingBlock.Uint64(), req.req.StartingBlock.Uint64()+128)
	}
	return str
}

func (q *syncQueue) stringifyResponseQueue() string {
	str := ""
	for _, resp := range q.responses {
		str = str + fmt.Sprintf("[start=%d end=%d] ", resp.start, resp.end)
	}
	return str
}

func (q *syncQueue) stop() {
	q.cancel()
}

// TODO: use this to determine who to try to sync from first
func (q *syncQueue) getSortedPeers() []*syncPeer { //nolint
	peers := make([]*syncPeer, len(q.peerScore))
	i := 0
	for pid, score := range q.peerScore {
		peers[i] = &syncPeer{
			pid:   pid,
			score: score,
		}
		i++
	}

	sort.Slice(peers, func(i, j int) bool {
		return peers[i].score < peers[j].score
	})

	return peers
}

func (q *syncQueue) pushBlockRequests(reqs []*BlockRequestMessage, pid peer.ID) {
	q.requestLock.Lock()
	defer q.requestLock.Unlock()

	var currLastRequestedBlock uint64
	if len(q.requests) > 0 {
		currLastRequestedBlock = q.requests[len(q.requests)-1].req.StartingBlock.Uint64()
	}

	for _, req := range reqs {
		// don't add requests that are already covered by existing requests
		if req.StartingBlock.Uint64() <= currLastRequestedBlock {
			continue
		}

		q.requests = append(q.requests, &syncRequest{
			pid: pid,
			req: req,
		})
	}
	sortRequests(q.requests)
}

func (q *syncQueue) pushBlockRequest(req *BlockRequestMessage, pid peer.ID) {
	q.requestLock.Lock()
	defer q.requestLock.Unlock()

	var currLastRequestedBlock uint64
	if len(q.requests) > 0 {
		currLastRequestedBlock = q.requests[len(q.requests)-1].req.StartingBlock.Uint64()
	}

	// don't add requests that are already covered by existing requests
	if req.StartingBlock.Uint64() <= currLastRequestedBlock {
		return
	}

	q.requests = append(q.requests, &syncRequest{
		pid: pid,
		req: req,
	})
	sortRequests(q.requests)
}

func (q *syncQueue) pushBlockResponse(resp *BlockResponseMessage, pid peer.ID) {
	if len(resp.BlockData) == 0 {
		return
	}

	start, end, err := resp.getStartAndEnd()
	if err != nil {
		logger.Debug("throwing away BlockResponseMessage as it doesn't contain block headers")
		return
	}

	q.responseLock.Lock()
	defer q.responseLock.Unlock()

	if len(q.responses) > 0 && start >= q.responses[0].start && end <= q.responses[len(q.responses)-1].end {
		logger.Debug("response is duplicate of others, discarding", "start", start, "end", end, "queue", q.stringifyResponseQueue())
		return
	}

	q.responses = append(q.responses, &syncResponse{
		pid:   pid,
		resp:  resp,
		start: start,
		end:   end,
	})

	logger.Debug("pushed block response to queue", "start", start, "end", end, "queue", q.stringifyResponseQueue())
	sortResponses(q.responses)
}

func (q *syncQueue) processBlockRequest() {
	for {
		select {
		case req := <-q.requestCh:
			if req.pid == "" {
				q.attemptSyncWithRandomPeer(req.req)
				continue
			}

			if err := q.beginSyncingWithPeer(req.pid, req.req); err != nil {
				logger.Debug("failed to send block request to peer, trying other peers", "peer", req.pid)
				q.attemptSyncWithRandomPeer(req.req)
			}
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) processBlockResponses() {
	for {
		select {
		case resp := <-q.responseCh:
			// TODO: change peerScore to sync.Map, currently this is the only place it's used
			q.peerScore[resp.pid]++
			logger.Debug("sending response to syncer", "start", resp.start, "end", resp.end)
			req := q.s.syncer.HandleBlockResponse(resp.resp)
			if req == nil {
				// we are done syncing
				q.unsetSyncingPeer(resp.pid)
				continue
			}

			q.pushBlockRequest(req, resp.pid)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) attemptSyncWithRandomPeer(req *BlockRequestMessage) {
	// TODO: when scoring is improved, try peers cached in syncQueue before random
	peers := q.s.host.peers()
	rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

	for _, peer := range peers {
		if err := q.beginSyncingWithPeer(peer, req); err == nil {
			logger.Debug("successfully began sync with peer", "peer", peer)
			return
		}
	}

	logger.Warn("failed to begin sync with any peer")
	// re-add request to queue, since we failed to send it to any peer
	q.pushBlockRequest(req, "")
}

func (q *syncQueue) beginSyncingWithPeer(peer peer.ID, req *BlockRequestMessage) error {
	q.syncingMu.Lock()
	defer q.syncingMu.Unlock()

	if _, syncing := q.syncing[peer]; syncing {
		return errors.New("already syncing with peer")
	}

	q.syncing[peer] = struct{}{}
	q.s.host.h.ConnManager().Protect(peer, "")

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

func sortRequests(reqs []*syncRequest) {
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].req.StartingBlock.Uint64() < reqs[j].req.StartingBlock.Uint64()
	})

	if len(reqs) <= 1 {
		return
	}

	i := 0
	for {
		if i >= len(reqs)-1 {
			return
		}

		if reqs[i].req.StartingBlock.Uint64() == reqs[i+1].req.StartingBlock.Uint64() && reflect.DeepEqual(reqs[i].req.Max, reqs[i+1].req.Max) {
			reqs = append(reqs[:i], reqs[i+1:]...)
		}

		i++
	}
}

func sortResponses(resps []*syncResponse) {
	sort.Slice(resps, func(i, j int) bool {
		return resps[i].start < resps[j].start
	})

	i := 0
	for {
		if i >= len(resps)-1 {
			return
		}

		if resps[i].start == resps[i+1].start && resps[i].end <= resps[i+1].end {
			resps = append(resps[:i], resps[i+1:]...)
		} else if resps[i].start == resps[i+1].start && resps[i].end > resps[i+1].end {
			if len(resps) == i+1 {
				resps = resps[:i+1]
				continue
			}

			resps = append(resps[:i+1], resps[i+2:]...)
		}

		i++
	}
}
