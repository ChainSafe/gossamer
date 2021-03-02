// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"context"
	"fmt"
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
func (s *Service) handleSyncMessage(stream libp2pnetwork.Stream, msg Message) error {
	if msg == nil {
		_ = stream.Close()
		return nil
	}

	// if it's a BlockRequest, call core for processing
	if req, ok := msg.(*BlockRequestMessage); ok {
		defer func() {
			_ = stream.Close()
		}()

		resp, err := s.syncer.CreateBlockResponse(req)
		if err != nil {
			logger.Trace("cannot create response for request")
			return nil
		}

		fmt.Println(resp)
		err = s.host.writeToStream(stream, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", stream.Conn().RemotePeer(), "error", err)
		}
	}

	return nil
}

var (
	blockRequestSize       uint32 = 128
	blockRequestQueueSize  int64  = 8
	maxBlockResponseSize   uint64 = 1024 * 1024 * 4 // 4mb
	badPeerThreshold       int    = -2
	protectedPeerThreshold int    = 10
)

type syncPeer struct {
	pid   peer.ID
	score int
}

type syncRequest struct {
	req *BlockRequestMessage
	to  peer.ID
}

type syncQueue struct {
	s         *Service
	ctx       context.Context
	cancel    context.CancelFunc
	peerScore *sync.Map // map[peer.ID]int; peers we have successfully synced from before -> their score; score increases on successful response

	requests  []*syncRequest // start block of message -> full message
	requestCh chan *syncRequest

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
		peerScore:   new(sync.Map),
		requests:    []*syncRequest{},
		requestCh:   make(chan *syncRequest),
		responses:   []*types.BlockData{},
		responseCh:  make(chan []*types.BlockData),
		benchmarker: newSyncBenchmarker(),
		buf:         make([]byte, maxBlockResponseSize),
	}
}

func (q *syncQueue) start() {
	go q.handleRequestQueue()
	go q.handleResponseQueue()

	go q.processBlockRequests()
	go q.processBlockResponses()

	go q.benchmark()
	go q.prunePeers()
}

func (q *syncQueue) handleRequestQueue() {
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

		logger.Trace("sync request queue", "queue", q.stringifyRequestQueue())
		head := q.requests[0]
		q.requests = q.requests[1:]
		q.requestCh <- head
	}
}

func (q *syncQueue) handleResponseQueue() {
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
			logger.Debug("response start is greater than head+1, waiting", "queue start", q.responses[0].Number().Int64(), "head+1", head.Int64()+1)
			q.responseLock.Unlock()

			q.setBlockRequests("")
			continue
		}

		logger.Trace("sync response queue", "queue", q.stringifyResponseQueue())
		q.responseLock.Unlock()
		q.responseCh <- q.responses
		q.responses = []*types.BlockData{}
	}
}

// prune peers with low score and connect to new peers
func (q *syncQueue) prunePeers() {
	for {
		time.Sleep(time.Second * 30)
		logger.Debug("✂️ pruning peers w/ low score...")

		peers := q.getSortedPeers()
		numPruned := 0

		for i := len(peers) - 1; i >= 0; i-- {
			// we're at our minimum peer count, don't disconnect from any more peers
			// we should discover more peers via dht between now and the next prune iteration
			if q.s.host.peerCount() <= q.s.cfg.MinPeers {
				break
			}

			// peers is a slice sorted from highest peer score to lowest, so we iterate backwards
			// until we reach peers that aren't low enough to be pruned
			if peers[i].score > badPeerThreshold {
				break
			}

			_ = q.s.host.closePeer(peers[i].pid)
			numPruned++
		}

		// protect peers with a high score so we don't disconnect from them
		numProtected := 0
		for i := 0; i < len(peers); i++ {
			if peers[i].score < protectedPeerThreshold {
				_ = q.s.host.cm.Unprotect(peers[i].pid, "")
				continue
			}

			q.s.host.cm.Protect(peers[i].pid, "")
			numProtected++
		}

		logger.Debug("✂️ finished pruning", "pruned count", numPruned, "protected count", numProtected, "peer count", q.s.host.peerCount())
	}
}

func (q *syncQueue) benchmark() {
	for {
		if q.ctx.Err() != nil {
			return
		}

		before, err := q.s.blockState.BestBlockHeader()
		if err != nil {
			logger.Error("failed to get best block header", "error", err)
			continue
		}

		if before.Number.Int64() >= q.goal {
			continue
		}

		q.benchmarker.begin(before.Number.Uint64())
		time.Sleep(time.Second * 5)

		after, err := q.s.blockState.BestBlockHeader()
		if err != nil {
			logger.Error("failed to get best block header", "error", err)
			continue
		}

		q.benchmarker.end(after.Number.Uint64())

		logger.Info("🔗 imported blocks", "from", before.Number, "to", after.Number,
			"hashes", fmt.Sprintf("[%s ... %s]", before.Hash(), after.Hash()),
		)

		logger.Info("🚣 currently syncing",
			"goal", q.goal,
			"average blocks/second", q.benchmarker.mostRecentAverage(),
			"overall average", q.benchmarker.average(),
		)
	}
}

func (q *syncQueue) stringifyRequestQueue() string {
	str := ""
	for _, req := range q.requests {
		if req == nil || req.req == nil || req.req.StartingBlock == nil {
			continue
		}

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
	peers := []*syncPeer{}

	q.peerScore.Range(func(pid, score interface{}) bool {
		peers = append(peers, &syncPeer{
			pid:   pid.(peer.ID),
			score: score.(int),
		})
		return true
	})

	sort.Slice(peers, func(i, j int) bool {
		return peers[i].score > peers[j].score
	})

	return peers
}

func (q *syncQueue) updatePeerScore(pid peer.ID, amt int) {
	score, ok := q.peerScore.Load(pid)
	if !ok {
		q.peerScore.Store(pid, amt)
	} else {
		q.peerScore.Store(pid, score.(int)+amt)
	}
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
	} else if len(q.responses) != 0 && q.responses[0].Number().Int64() == q.currEnd+1 {
		// we have some responses queued, and the next block data is equal to the data we're currently syncing + 1
		start = q.responses[len(q.responses)-1].Number().Int64()
	} else {
		// we aren't syncing anything and don't have anything queued
		start = head.Int64() + 1
	}

	logger.Trace("setting block request queue", "start", start, "goal", q.goal)

	reqs := createBlockRequests(start, q.goal)

	newReqs := []*syncRequest{}
	for _, req := range reqs {
		newReqs = append(newReqs, &syncRequest{
			to:  to,
			req: req,
		})
	}
	newReqs = sortRequests(newReqs)
	q.requests = newReqs

	logger.Trace("sync request queue", "queue", q.stringifyRequestQueue())
}

func (q *syncQueue) pushBlockResponse(resp *BlockResponseMessage, pid peer.ID) {
	if len(resp.BlockData) == 0 {
		return
	}

	head, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Error("failed to get best block number", "error", err)
		return
	}

	start, end, err := resp.getStartAndEnd()
	if err != nil {
		logger.Trace("throwing away BlockResponseMessage as it doesn't contain block headers")
		// update peer's score
		q.updatePeerScore(pid, -1)
		return
	}

	if resp.BlockData[0].Body == nil || !resp.BlockData[0].Body.Exists() {
		logger.Trace("throwing away BlockResponseMessage as it doesn't contain block bodies")
		// update peer's score
		q.updatePeerScore(pid, -1)
		return
	}

	// update peer's score
	q.updatePeerScore(pid, 3)

	q.responseLock.Lock()
	defer q.responseLock.Unlock()

	for _, bd := range resp.BlockData {
		if bd.Number() == nil || bd.Number().Int64() < head.Int64() {
			continue
		}

		q.responses = append(q.responses, bd)
	}

	q.responses = sortResponses(q.responses)
	logger.Debug("pushed block data to queue", "start", start, "end", end, "queue", q.stringifyResponseQueue())
}

func (q *syncQueue) processBlockRequests() {
	for {
		select {
		case req := <-q.requestCh:
			q.trySync(req)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) trySync(req *syncRequest) {
	if q.ctx.Err() != nil {
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
		q.updatePeerScore(req.to, -1)
	}

	logger.Debug("trying peers in prioritized order...")
	syncPeers := q.getSortedPeers()

	for _, peer := range syncPeers {
		// if peer doesn't respond multiple times, then ignore them TODO: determine best values for this
		if peer.score <= badPeerThreshold {
			break
		}

		resp, err := q.syncWithPeer(peer.pid, req.req)
		if err != nil {
			logger.Trace("failed to sync with peer", "peer", peer.pid, "error", err)
			q.updatePeerScore(peer.pid, -1)
			continue
		}

		q.pushBlockResponse(resp, peer.pid)
		return
	}

	logger.Debug("failed to sync with any peer :(")
}

func (q *syncQueue) syncWithPeer(peer peer.ID, req *BlockRequestMessage) (*BlockResponseMessage, error) {
	fullSyncID := q.s.host.protocolID + syncID

	q.s.host.h.ConnManager().Protect(peer, "")
	defer q.s.host.h.ConnManager().Unprotect(peer, "")
	defer q.s.host.closeStream(peer, fullSyncID)

	ctx, cancel := context.WithTimeout(q.ctx, time.Second*2)
	defer cancel()

	s, err := q.s.host.h.NewStream(ctx, peer, fullSyncID)
	if err != nil {
		return nil, err
	}

	err = q.s.host.writeToStream(s, req)
	if err != nil {
		return nil, err
	}

	return q.receiveBlockResponse(s)
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

func (q *syncQueue) processBlockResponses() {
	for {
		select {
		case data := <-q.responseCh:
			bestNum, err := q.s.blockState.BestBlockNumber()
			if err != nil {
				panic(err)
			}

			if data[len(data)-1].Number().Int64() <= bestNum.Int64() {
				logger.Debug("ignoring block data that is below our head", "got", data[len(data)-1].Number().Int64(), "head", bestNum.Int64())
				q.currStart = 0
				q.currEnd = 0
				continue
			}

			q.currStart = data[0].Number().Int64()
			q.currEnd = data[len(data)-1].Number().Int64()
			logger.Debug("sending block data to syncer", "start", q.currStart, "end", q.currEnd)

			err = q.s.syncer.ProcessBlockData(data)
			if err != nil {
				logger.Warn("failed to handle block data; re-adding to queue", "start", q.currStart, "end", q.currEnd, "error", err)
				q.currStart = 0
				q.currEnd = 0
				q.setBlockRequests("")
				continue
			}

			q.currStart = 0
			q.currEnd = 0
		case <-q.ctx.Done():
			return
		}
	}
}

// handleBlockAnnounceHandshake handles a block that a peer claims to have through a HandleBlockAnnounceHandshake
func (q *syncQueue) handleBlockAnnounceHandshake(blockNum uint32, from peer.ID) {
	q.updatePeerScore(from, 1)

	bestNum, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Error("failed to get best block number", "error", err)
		return
	}

	if bestNum.Int64() >= int64(blockNum) || q.goal >= int64(blockNum) {
		return
	}

	q.goal = int64(blockNum)
	q.setBlockRequests(from)
}

func (q *syncQueue) handleBlockAnnounce(msg *BlockAnnounceMessage, from peer.ID) {
	q.updatePeerScore(from, 1)

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
