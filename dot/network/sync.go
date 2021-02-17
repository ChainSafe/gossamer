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

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

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
		_ = stream.Close()
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
		s.host.closeStream(peer, syncID)
		return nil
	}

	if resp, ok := msg.(*BlockResponseMessage); ok {
		if isSyncing := s.syncQueue.isSyncing(peer); !isSyncing {
			logger.Debug("not currently syncing with peer", "peer", peer)
			s.host.closeStream(peer, syncID)
			return nil
		}

		s.syncQueue.pushBlockResponse(resp, peer)
	}

	// if it's a BlockRequest, call core for processing
	if req, ok := msg.(*BlockRequestMessage); ok {
		resp, err := s.syncer.CreateBlockResponse(req)
		if err != nil {
			logger.Debug("cannot create response for request")
			s.host.closeStream(peer, syncID)
			return nil
		}

		err = s.host.send(peer, syncID, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", peer)
			s.host.closeStream(peer, syncID)
		}
	}

	return nil
}

var (
	blockRequestSize      uint32 = 128
	blockRequestQueueSize int64  = 5
	blockDataQueueSize    int64  = 256
)

type syncPeer struct { //nolint
	pid   peer.ID
	score int
}

type syncRequest struct {
	req *BlockRequestMessage
	pid peer.ID // rename to "to"?
}

// type syncResponse struct {
// 	data       *types.BlockData
// 	pid        peer.ID // rename to "from"?
// }

type blockRange struct {
	start, end int64
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

	responses    []*types.BlockData
	responseCh   chan []*types.BlockData
	responseLock sync.RWMutex

	waitingForResp     *blockRange
	gotRespCh          chan *blockRange
	goal               int64 // goal block number we are trying to sync to
	currStart, currEnd int64 // the start and end of the BlockResponse we are currently handling; 0 and 0 if we are not currently handling any

	benchmarker *syncBenchmarker
}

func newSyncQueue(s *Service) *syncQueue {
	ctx, cancel := context.WithCancel(s.ctx)

	return &syncQueue{
		s:           s,
		ctx:         ctx,
		cancel:      cancel,
		peerScore:   make(map[peer.ID]int),
		syncing:     make(map[peer.ID]struct{}),
		requests:    []*syncRequest{},
		requestCh:   make(chan *syncRequest),
		responses:   []*types.BlockData{},
		responseCh:  make(chan []*types.BlockData),
		gotRespCh:   make(chan *blockRange),
		benchmarker: newSyncBenchmarker(),
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

			// head, err := q.s.blockState.BestBlockNumber()
			// if err != nil {
			// 	continue
			// }

			// if we have block requests to send, put them into requestCh
			q.requestLock.Lock()
			if len(q.requests) == 0 {
				//if q.goal == head.Int64() || q.goal == 0 {
				q.requestLock.Unlock()
				continue
				// }

				// q.setBlockRequests("")
			}

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

			q.responseLock.Lock()
			if len(q.responses) == 0 {
				q.responseLock.Unlock()

				// if head.Int64() < q.goal {
				// 	q.requestLock.Lock()
				// 	if len(q.requests) == 0 && !q.waitingForResp {
				// 		q.setBlockRequests("")
				// 	}
				// 	q.requestLock.Unlock()
				// }
				continue
			}

			if q.responses[0].Number().Int64() > head.Int64()+1 {
				logger.Debug("response start isn't head+1, waiting", "queue start", q.responses[0].Number().Int64(), "head+1", head.Int64()+1)
				q.responseLock.Unlock()

				// q.requestLock.Lock()
				// if len(q.requests) == 0 && !q.waitingForResp {
				// 	q.setBlockRequests("")
				// }
				// q.requestLock.Unlock()
				continue
			}

			logger.Debug("sync response queue", "queue", q.stringifyResponseQueue())
			q.responseLock.Unlock()
			q.responseCh <- q.responses
			q.responses = []*types.BlockData{}
		}
	}()

	go q.processBlockRequests()
	go q.processBlockResponses()
	go q.benchmark()
}

func (q *syncQueue) benchmark() {
	for {
		head, err := q.s.blockState.BestBlockNumber()
		if err != nil {
			logger.Error("failed to get best block number", "error", err)
			return // TODO: handle this / panic?
		}

		q.benchmarker.begin(head.Uint64())
		time.Sleep(time.Second * 15)

		head, err = q.s.blockState.BestBlockNumber()
		if err != nil {
			logger.Error("failed to get best block number", "error", err)
			return // TODO: handle this / panic?
		}

		q.benchmarker.end(head.Uint64())
		avg := q.benchmarker.average()
		logger.Info("🚣 currently syncing", "average blocks/second", avg)
	}
}

func (q *syncQueue) stringifyRequestQueue() string {
	str := ""
	for _, req := range q.requests {
		str = str + fmt.Sprintf("[start=%d end=%d] ", req.req.StartingBlock.Uint64(), req.req.StartingBlock.Uint64()+128)
	}
	return str
}

func (q *syncQueue) stringifyResponseQueue() string {
	if len(q.responses) == 0 {
		return "[empty]"
	}
	return fmt.Sprintf("[start=%d end=%d] ", q.responses[0].Number().Int64(), q.responses[len(q.responses)-1].Number().Int64())
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

func (q *syncQueue) setBlockRequests(pid peer.ID) {
	head, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		return
	}

	var start int64
	// we are currently syncing some blocks, don't have any other blocks to process queued
	if q.currEnd != 0 && len(q.responses) == 0 {
		start = q.currEnd + 1
	} else if len(q.responses) != 0 && q.responses[0].Number().Int64() <= q.currEnd {
		// we have some responses queued, and the next
		start = q.responses[len(q.responses)-1].Number().Int64()
	} else {
		// we aren't syncing anything and don't have anything queued
		start = head.Int64() + 1
	}

	logger.Debug("setting block request queue", "start", start, "goal", q.goal)

	reqs := createBlockRequests(start, q.goal)

	// q.requestLock.Lock()
	// defer q.requestLock.Unlock()

	//var currLastRequestedBlock uint64
	// if len(q.requests) > 0 {
	// 	currLastRequestedBlock = q.requests[len(q.requests)-1].req.StartingBlock.Uint64()
	// }
	//

	q.requests = []*syncRequest{}
	for _, req := range reqs {
		// if req.StartingBlock.Uint64() + uint64(blockRequestSize) <= uint64(q.currEnd) {
		// 	return
		// }

		// // // don't add requests that are already covered by existing requests
		// // if req.StartingBlock.Uint64() <= currLastRequestedBlock {
		// // 	continue
		// // }
		// if req.StartingBlock.Uint64() + uint64(blockRequestSize) > q.requestedTo {
		// 	q.requestedTo = req.StartingBlock.Uint64() + uint64(blockRequestSize)
		// }

		q.requests = append(q.requests, &syncRequest{
			pid: pid,
			req: req,
		})
	}
	sortRequests(q.requests)
	logger.Debug("sync request queue", "queue", q.stringifyRequestQueue())
}

// func (q *syncQueue) pushBlockRequest(req *BlockRequestMessage, pid peer.ID) {
// 	q.requestLock.Lock()
// 	defer q.requestLock.Unlock()

// 	var currLastRequestedBlock uint64
// 	if len(q.requests) > 0 {
// 		currLastRequestedBlock = q.requests[len(q.requests)-1].req.StartingBlock.Uint64()
// 	}

// 	// don't add requests that are already covered by existing requests
// 	if req.StartingBlock.Uint64() <= currLastRequestedBlock {
// 		return
// 	}

// 	// reject request that oerlaps with blocks we are currently syncing
// 	if req.StartingBlock.Uint64() <= uint64(q.currEnd) {
// 		return
// 	}

// 	if req.StartingBlock.Uint64() + uint64(blockRequestSize) > q.requestedTo {
// 		q.requestedTo = req.StartingBlock.Uint64() + uint64(blockRequestSize)
// 	}

// 	q.requests = append(q.requests, &syncRequest{
// 		pid: pid,
// 		req: req,
// 	})
// 	sortRequests(q.requests)
// }

func (q *syncQueue) pushBlockResponse(resp *BlockResponseMessage, pid peer.ID) {
	//q.waitingForResp = nil

	if len(resp.BlockData) == 0 || len(q.responses)+len(resp.BlockData) >= int(blockDataQueueSize) {
		return
	}

	head, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Error("failed to get best block number", "error", err)
		return // TODO: handle this / panic?
	}

	start, end, err := resp.getStartAndEnd()
	if err != nil {
		logger.Debug("throwing away BlockResponseMessage as it doesn't contain block headers")
		return
	}

	q.responseLock.Lock()

	for _, bd := range resp.BlockData {
		if bd.Number() == nil || bd.Number().Int64() < head.Int64() {
			continue
		}

		q.responses = append(q.responses, bd)
	}
	q.responseLock.Unlock()

	// TODO: change peerScore to sync.Map, currently this is the only place it's used
	q.peerScore[pid]++
	q.gotRespCh <- &blockRange{
		start: start,
		end:   end,
	}

	q.responseLock.Lock()
	defer q.responseLock.Unlock()

	sortResponses(q.responses)
	logger.Debug("pushed block data to queue", "start", start, "end", end, "queue", q.stringifyResponseQueue())
}

func (q *syncQueue) processBlockRequests() {
	for {
		select {
		case req := <-q.requestCh:
			//q.waitingForResp = true
			q.ensureResponseReceived(req)

			// if len(req.pid) == 0 {
			// 	q.attemptSyncWithRandomPeer(req.req)
			// 	continue
			// }

			// if err := q.beginSyncingWithPeer(req.pid, req.req); err != nil {
			// 	q.unsetSyncingPeer(req.pid)
			// 	logger.Debug("failed to send block request to peer, trying other peers", "peer", req.pid)
			// 	q.attemptSyncWithRandomPeer(req.req)
			// }
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) ensureResponseReceived(req *syncRequest) {
	var attemptToSyncFunc func(*BlockRequestMessage) = q.attemptSyncWithPreferedPeers

	for {
		logger.Debug("beginning to send out request", "start", req.req.StartingBlock.Uint64())
		if len(req.pid) == 0 {
			attemptToSyncFunc(req.req)
		} else {

			if err := q.beginSyncingWithPeer(req.pid, req.req); err != nil {
				q.unsetSyncingPeer(req.pid)
				logger.Debug("failed to send block request to peer, trying other peers", "peer", req.pid)
				attemptToSyncFunc(req.req)
			}

		}

		select {
		case resp := <-q.gotRespCh:
			if resp.start != int64(req.req.StartingBlock.Uint64()) {
				logger.Error("received response that we didn't request!!")
				continue
			}

			logger.Debug("response received", "start", resp.start, "end", resp.end)
			return
		case <-time.After(time.Second * 5):
			logger.Warn("haven't received a response in a while...", "start", req.req.StartingBlock.Uint64())
			attemptToSyncFunc = q.attemptSyncWithRandomPeer
			continue
		}
	}
}

func (q *syncQueue) processBlockResponses() {
	for {
		select {
		case data := <-q.responseCh:
			//q.waitingForResp = false

			q.currStart = data[0].Number().Int64()
			q.currEnd = data[len(data)-1].Number().Int64()
			logger.Debug("sending block data to syncer", "start", q.currStart, "end", q.currEnd)

			err := q.s.syncer.ProcessBlockData(data)
			q.currStart = 0
			q.currEnd = 0
			if err != nil {
				logger.Error("failed to handle block data; re-adding to queue", "start", q.currStart, "end", q.currEnd, "error", err)
				q.setBlockRequests("")
				continue
			}

			//q.unsetSyncingPeer(pid)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) attemptSyncWithPreferedPeers(req *BlockRequestMessage) {
	var peers []peer.ID
	for _, p := range q.getSortedPeers() {
		peers = append(peers, p.pid)
	}

	q.attemptSyncWithPeers(peers, req)
}

func (q *syncQueue) attemptSyncWithRandomPeer(req *BlockRequestMessage) {
	// var peers []peer.ID
	// for pid := range q.peerScore {
	// 	peers = append(peers, pid)
	// }

	// TODO: when scoring is improved, try peers cached in syncQueue before random
	//if len(peers) == 0 {
	peers := q.s.host.peers()
	//}
	rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

	q.attemptSyncWithPeers(peers, req)
}

func (q *syncQueue) attemptSyncWithPeers(peers []peer.ID, req *BlockRequestMessage) {
	for _, peer := range peers {
		if err := q.beginSyncingWithPeer(peer, req); err == nil {
			logger.Debug("successfully sent BlockRequest to peer", "peer", peer)
			return
		} else {
			q.unsetSyncingPeer(peer)
		}
	}

	logger.Warn("failed to begin sync with any peer")
	// re-add request to queue, since we failed to send it to any peer
	//q.pushBlockRequest(req, "")
	q.setBlockRequests("")
	//q.waitingForResp = true
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

func sortResponses(resps []*types.BlockData) {
	sort.Slice(resps, func(i, j int) bool {
		return resps[i].Number().Int64() < resps[j].Number().Int64()
	})

	hasData := make(map[common.Hash]struct{})

	i := 0
	for {
		if i >= len(resps)-1 {
			return
		}

		if _, has := hasData[resps[i].Hash]; !has {
			hasData[resps[i].Hash] = struct{}{}
			i++
		} else if has {
			resps = append(resps[:i], resps[i+1:]...)
		}
	}
}

// handleBlockAnnounceHandshake handles a block that a peer claims to have through a HandleBlockAnnounceHandshake
func (q *syncQueue) handleBlockAnnounceHandshake(blockNum uint32, from peer.ID) {
	// if len(q.requests) > int(blockRequestQueueSize) {
	// 	return
	// }

	q.peerScore[from]++

	bestNum, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Error("failed to get best block number", "error", err)
		return // TODO: handle this / panic?
	}

	if bestNum.Int64() >= int64(blockNum) || q.goal >= int64(blockNum) {
		return
	}

	q.goal = int64(blockNum)

	// if len(q.requests) >= int(blockRequestQueueSize) {
	// 	return
	// }

	// var start int64
	// if q.currEnd != 0 && len(q.responses) == 0 {
	// 	start = q.currEnd + 1
	// } else if len(q.responses) != 0 {
	// 	start = q.responses[len(q.responses)-1].Number().Int64()
	// } else {
	// 	start = bestNum.Int64() + 1
	// }

	// reqs := createBlockRequests(start, q.goal)
	// q.pushBlockRequests(reqs, from)
	q.setBlockRequests(from)
}

func (q *syncQueue) handleBlockAnnounce(msg *BlockAnnounceMessage, from peer.ID) {
	q.peerScore[from]++

	// if len(q.requests) > int(blockRequestQueueSize) {
	// 	return
	// }

	// create block request to send
	// bestNum, err := q.s.blockState.BestBlockNumber() //nolint
	// if err != nil {
	// 	logger.Error("failed to get best block number", "error", err)
	// 	return
	// }

	header, err := types.NewHeader(
		msg.ParentHash,
		msg.Number,
		msg.StateRoot,
		msg.ExtrinsicsRoot,
		msg.Digest,
	)
	if err != nil {
		logger.Error("failed to create header from BlockAnnounce", "error", err)
		return
	}

	has, _ := q.s.blockState.HasBlockBody(header.Hash())
	if has {
		return
	}

	if header.Number.Int64() > q.goal {
		q.goal = header.Number.Int64()
	}

	// if len(q.requests) >= int(blockRequestQueueSize) {
	// 	return
	// }

	// if we already have blocks up to the BlockAnnounce number, only request the block in the BlockAnnounce
	// var start int64
	// if bestNum.Cmp(header.Number) > 0 {
	// 	start = header.Number.Int64()
	// } else if q.currEnd != 0 && len(q.responses) == 0 {
	// 	start = q.currEnd + 1
	// } else if len(q.responses) != 0 {
	// 	start = q.responses[len(q.responses)-1].Number().Int64()
	// } else {
	// 	start = bestNum.Int64() + 1
	// }

	// reqs := createBlockRequests(start, q.goal)
	// q.pushBlockRequests(reqs, from)
	q.setBlockRequests(from)
}

func createBlockRequests(start, end int64) []*BlockRequestMessage {
	numReqs := (end - start) / int64(blockRequestSize)
	if numReqs > blockRequestQueueSize {
		numReqs = blockRequestQueueSize
	}

	if end-start < int64(blockRequestSize) {
		numReqs = 1
	}

	reqs := make([]*BlockRequestMessage, numReqs)
	for i := 0; i < int(numReqs); i++ {
		offset := i * int(blockRequestSize)
		reqs[i] = createBlockRequest(start+int64(offset), blockRequestSize)
	}
	return reqs
}

func createBlockRequest(startInt int64, size uint32) *BlockRequestMessage {
	start, _ := variadic.NewUint64OrHash(uint64(startInt))
	//logger.Debug("creating block request", "start", start)

	blockRequest := &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0, // ascending
		Max:           optional.NewUint32(true, size),
	}

	return blockRequest
}
