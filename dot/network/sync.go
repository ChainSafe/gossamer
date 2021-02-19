package network

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

	// if it's a BlockRequest, call core for processing
	if req, ok := msg.(*BlockRequestMessage); ok {
		defer s.host.closeStream(peer, syncID)

		resp, err := s.syncer.CreateBlockResponse(req)
		if err != nil {
			logger.Debug("cannot create response for request")
			return nil
		}

		err = s.host.send(peer, syncID, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", peer)
		}
	}

	return nil
}

var (
	blockRequestSize      uint32 = 128
	blockRequestQueueSize int64  = 8
	blockDataQueueSize    int64  = blockRequestQueueSize * int64(blockRequestSize)
	maxBlockResponseSize  uint64 = 1024 * 256
)

type syncPeer struct {
	pid   peer.ID
	score int
}

type syncRequest struct {
	req *BlockRequestMessage
	to  peer.ID
}

type blockRange struct {
	start, end int64
	from       peer.ID
}

type syncQueue struct {
	s         *Service
	ctx       context.Context
	cancel    context.CancelFunc
	peerScore map[peer.ID]int // peers we have successfully synced from before -> their score; score increases on successful response; decreases otherwise.

	requests    []*syncRequest // start block of message -> full message
	requestCh   chan *syncRequest
	requestLock sync.RWMutex

	responses    []*types.BlockData
	responseCh   chan []*types.BlockData
	responseLock sync.RWMutex

	buf                []byte
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
		requests:    []*syncRequest{},
		requestCh:   make(chan *syncRequest),
		responses:   []*types.BlockData{},
		responseCh:  make(chan []*types.BlockData),
		benchmarker: newSyncBenchmarker(),
		buf:         make([]byte, maxBlockResponseSize),
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
			q.requestLock.Lock()
			if len(q.requests) == 0 {
				q.requestLock.Unlock()
				continue
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
				continue
			}

			if q.responses[0].Number().Int64() > head.Int64()+1 {
				logger.Debug("response start isn't head+1, waiting", "queue start", q.responses[0].Number().Int64(), "head+1", head.Int64()+1)
				q.responseLock.Unlock()

				q.requestLock.Lock()
				q.setBlockRequests("")
				q.requestLock.Unlock()
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
		logger.Info("🚣 currently syncing", "average blocks/second", q.benchmarker.mostRecentAverage(), "overall average", q.benchmarker.average())
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

// getSortedPeers is used to determine who to try to sync from first
func (q *syncQueue) getSortedPeers() []*syncPeer {
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

func (q *syncQueue) setBlockRequests(to peer.ID) {
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

	q.requests = []*syncRequest{}
	for _, req := range reqs {
		q.requests = append(q.requests, &syncRequest{
			to:  to,
			req: req,
		})
	}
	q.requests = sortRequests(q.requests)
	logger.Debug("sync request queue", "queue", q.stringifyRequestQueue())
}

func (q *syncQueue) pushBlockResponse(resp *BlockResponseMessage, pid peer.ID) {
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

	// TODO: change peerScore to sync.Map
	q.peerScore[pid]++

	q.responseLock.Lock()
	defer q.responseLock.Unlock()

	q.responses = sortResponses(q.responses)
	logger.Debug("pushed block data to queue", "start", start, "end", end, "queue", q.stringifyResponseQueue())
}

func (q *syncQueue) processBlockRequests() {
	for {
		select {
		case req := <-q.requestCh:
			q.ensureResponseReceived(req)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) ensureResponseReceived(req *syncRequest) {
	numRetries := 3
	i := 0

	for {
		if q.ctx.Err() != nil {
			return
		}

		if i == numRetries {
			logger.Error("failed to sync with any peer after 3 retries")
			return
		}

		logger.Debug("beginning to send out request", "start", req.req.StartingBlock.Uint64())
		if len(req.to) != 0 {
			resp, err := q.syncWithPeer(req.to, req.req)
			if err == nil {
				q.pushBlockResponse(resp, req.to)
				return
			}

			logger.Debug("failed to sync with peer", "peer", req.to, "error", err)
		}

		// try highest scored peers first
		var peers []peer.ID
		for _, p := range q.getSortedPeers() {
			peers = append(peers, p.pid)
		}

		for _, peer := range peers {
			resp, err := q.syncWithPeer(peer, req.req)
			if err != nil {
				logger.Debug("failed to sync with peer", "peer", peer, "error", err)
				continue
			}

			q.pushBlockResponse(resp, peer)
			return
		}

		logger.Debug("failed to sync with preferred peers, trying random...")

		peers = q.s.host.peers()
		rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

		for _, peer := range peers {
			resp, err := q.syncWithPeer(peer, req.req)
			if err != nil {
				logger.Debug("failed to sync with peer", "peer", peer, "error", err)
				continue
			}

			q.pushBlockResponse(resp, peer)
			return
		}

		logger.Warn("failed to sync with any peer :(")
	}
}

func (q *syncQueue) receiveBlockResponse(stream libp2pnetwork.Stream) (*BlockResponseMessage, error) {
	n, err := readStream(stream, q.buf)
	if err != nil {
		return nil, err
	}

	msg := new(BlockResponseMessage)
	err = msg.Decode(q.buf[:n])
	return msg, err
}

// readStream reads from the stream into the given buffer, returning the number of bytes read
func readStream(stream libp2pnetwork.Stream, buf []byte) (int, error) {
	r := bufio.NewReader(stream)

	var (
		tot int
	)

	length, err := readLEB128ToUint64(r)
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, err // TODO: read bytes read from readLEB128ToUint64
	}

	if length == 0 {
		return 0, err // TODO: read bytes read from readLEB128ToUint64
	}

	// TODO: check if length > len(buf), if so probably log.Crit
	if length > maxBlockResponseSize {
		logger.Warn("received message with size greater than maxBlockResponseSize, discarding", "length", length)
		for {
			_, err = r.Discard(int(maxBlockResponseSize))
			if err != nil {
				break
			}
		}
		return 0, fmt.Errorf("message size greater than maximum: got %d", length)
	}

	tot = 0
	for i := 0; i < maxReads; i++ {
		n, err := r.Read(buf[tot:])
		if err != nil {
			return n + tot, err
		}

		tot += n
		if tot == int(length) {
			break
		}
	}

	if tot != int(length) {
		return tot, fmt.Errorf("failed to read entire message: expected %d bytes", length)
	}

	return tot, nil
}

func (q *syncQueue) processBlockResponses() {
	for {
		select {
		case data := <-q.responseCh:
			q.currStart = data[0].Number().Int64()
			q.currEnd = data[len(data)-1].Number().Int64()
			logger.Debug("sending block data to syncer", "start", q.currStart, "end", q.currEnd)

			err := q.s.syncer.ProcessBlockData(data)
			q.currStart = 0
			q.currEnd = 0
			if err != nil {
				logger.Warn("failed to handle block data; re-adding to queue", "start", q.currStart, "end", q.currEnd, "error", err)
				q.setBlockRequests("")
				continue
			}
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) syncWithPeer(peer peer.ID, req *BlockRequestMessage) (*BlockResponseMessage, error) {
	fullSyncID := q.s.host.protocolID + syncID

	q.s.host.h.ConnManager().Protect(peer, "")
	defer q.s.host.h.ConnManager().Unprotect(peer, "")
	defer q.s.host.closeStream(peer, fullSyncID)

	s, err := q.s.host.h.NewStream(q.ctx, peer, fullSyncID)
	if err != nil {
		return nil, err
	}

	err = q.s.host.writeToStream(s, req)
	if err != nil {
		return nil, err
	}

	return q.receiveBlockResponse(s)
}

// handleBlockAnnounceHandshake handles a block that a peer claims to have through a HandleBlockAnnounceHandshake
func (q *syncQueue) handleBlockAnnounceHandshake(blockNum uint32, from peer.ID) {
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
	q.setBlockRequests(from)
}

func (q *syncQueue) handleBlockAnnounce(msg *BlockAnnounceMessage, from peer.ID) {
	q.peerScore[from]++

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

	if header.Number.Int64() <= q.goal {
		return
	}

	q.goal = header.Number.Int64()
	q.setBlockRequests(from)
}

func createBlockRequests(start, end int64) []*BlockRequestMessage {
	if start > end {
		return nil
	}

	numReqs := (end - start) / int64(blockRequestSize)
	if numReqs > blockRequestQueueSize {
		numReqs = blockRequestQueueSize
	}

	if end-start < int64(blockRequestSize) {
		// +1 because we want to include the block w/ the ending number
		req := createBlockRequest(start, uint32(end-start)+1)
		return []*BlockRequestMessage{req}
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

	blockRequest := &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0, // ascending
		Max:           optional.NewUint32(true, size),
	}

	return blockRequest
}

func sortRequests(reqs []*syncRequest) []*syncRequest {
	if len(reqs) == 0 {
		return reqs
	}

	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].req.StartingBlock.Uint64() < reqs[j].req.StartingBlock.Uint64()
	})

	i := 0
	for {
		if i >= len(reqs)-1 {
			return reqs
		}

		if reqs[i].req.StartingBlock.Uint64() == reqs[i+1].req.StartingBlock.Uint64() && reflect.DeepEqual(reqs[i].req.Max, reqs[i+1].req.Max) {
			reqs = append(reqs[:i], reqs[i+1:]...)
		}

		i++
	}
}

func sortResponses(resps []*types.BlockData) []*types.BlockData {
	sort.Slice(resps, func(i, j int) bool {
		return resps[i].Number().Int64() < resps[j].Number().Int64()
	})

	hasData := make(map[common.Hash]struct{})

	i := 0
	for {
		if i > len(resps)-1 {
			return resps
		}

		if _, has := hasData[resps[i].Hash]; !has {
			hasData[resps[i].Hash] = struct{}{}
			i++
		} else if has {
			resps = append(resps[:i], resps[i+1:]...)
		}
	}
}
