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

		err = s.host.writeToStream(stream, resp)
		if err != nil {
			logger.Error("failed to send BlockResponse message", "peer", stream.Conn().RemotePeer(), "error", err)
		}
	}

	return nil
}

var (
	blockRequestSize        uint32 = 128
	blockRequestBufferSize  int    = 6
	blockResponseBufferSize int    = 6

	maxBlockResponseSize   uint64 = 1024 * 1024 * 4 // 4mb
	badPeerThreshold       int    = -2
	protectedPeerThreshold int    = 7

	defaultSlotDuration = time.Second * 6
)

var (
	errEmptyResponseData      = fmt.Errorf("response data is empty")
	errEmptyJustificationData = fmt.Errorf("no justifications in response data")
)

type syncPeer struct {
	pid   peer.ID
	score int
}

type syncRequest struct {
	req *BlockRequestMessage
	to  peer.ID
}

type requestData struct {
	sent     bool // if the request has been already sent to all peers
	received bool
	from     peer.ID
}

type syncQueue struct {
	s            *Service
	slotDuration time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	peerScore    *sync.Map // map[peer.ID]int; peers we have successfully synced from before -> their score; score increases on successful response

	requestData              *sync.Map // map[uint64]requestData; map of start # of request -> requestData
	justificationRequestData *sync.Map // map[common.Hash]requestData; map of requests of justifications -> requestData
	requestCh                chan *syncRequest

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
		s:                        s,
		slotDuration:             defaultSlotDuration,
		ctx:                      ctx,
		cancel:                   cancel,
		peerScore:                new(sync.Map),
		requestData:              new(sync.Map),
		justificationRequestData: new(sync.Map),
		requestCh:                make(chan *syncRequest, blockRequestBufferSize),
		responses:                []*types.BlockData{},
		responseCh:               make(chan []*types.BlockData, blockResponseBufferSize),
		benchmarker:              newSyncBenchmarker(),
		buf:                      make([]byte, maxBlockResponseSize),
	}
}

func (q *syncQueue) start() {
	go q.handleResponseQueue()
	go q.syncAtHead()
	go q.finalizeAtHead()

	go q.processBlockRequests()
	go q.processBlockResponses()

	go q.benchmark()
	go q.prunePeers()
}

func (q *syncQueue) syncAtHead() {
	prev, err := q.s.blockState.BestBlockHeader()
	if err != nil {
		logger.Error("failed to get best block header", "error", err)
		return
	}

	q.s.syncer.SetSyncing(true)

	for {
		select {
		// sleep for average block time TODO: make this configurable from slot duration
		case <-time.After(q.slotDuration):
		case <-q.ctx.Done():
			return
		}

		curr, err := q.s.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		// we aren't at the head yet, sleep
		if curr.Number.Int64() < q.goal && curr.Number.Cmp(prev.Number) > 0 {
			prev = curr
			continue
		}

		q.s.syncer.SetSyncing(false)

		// we have received new blocks since the last check, sleep
		if prev.Number.Int64() < curr.Number.Int64() {
			prev = curr
			continue
		}

		prev = curr
		start := uint64(curr.Number.Int64()) + 1
		logger.Debug("haven't received new blocks since last check, pushing request", "start", start)
		q.requestData.Delete(start)
		q.pushRequest(start, 1, "")
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
		if err != nil || head == nil {
			continue
		}

		q.responseLock.Lock()
		if len(q.responses) == 0 {
			q.responseLock.Unlock()

			if len(q.requestCh) == 0 && head.Int64() < q.goal {
				q.pushRequest(uint64(head.Int64()+1), blockRequestBufferSize, "")
			}
			continue
		}

		start := q.responses[0].Number()
		if start == nil {
			q.responseLock.Unlock()
			continue
		}

		if start.Int64() > head.Int64()+1 {
			logger.Debug("response start is greater than head+1, waiting", "queue start", start.Int64(), "head+1", head.Int64()+1)
			q.responseLock.Unlock()

			q.pushRequest(uint64(head.Int64()+1), 1, "")
			continue
		}

		logger.Debug("pushing to response queue", "start", start)
		q.responseCh <- q.responses
		logger.Debug("pushed responses!", "start", start)
		q.responses = []*types.BlockData{}
		q.responseLock.Unlock()
	}
}

// prune peers with low score and connect to new peers
func (q *syncQueue) prunePeers() {
	for {
		select {
		case <-time.After(time.Second * 30):
		case <-q.ctx.Done():
			return
		}

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
			continue
		}

		if before.Number.Int64() >= q.goal {
			finalized, err := q.s.blockState.GetFinalizedHeader(0, 0) //nolint
			if err != nil {
				continue
			}

			logger.Info("💤 node waiting", "head", before.Number, "finalized", finalized.Number)
			time.Sleep(time.Second * 5)
			continue
		}

		q.benchmarker.begin(before.Number.Uint64())
		time.Sleep(time.Second * 5)

		after, err := q.s.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		q.benchmarker.end(after.Number.Uint64())

		logger.Info("🔗 imported blocks", "from", before.Number, "to", after.Number,
			"hashes", fmt.Sprintf("[%s ... %s]", before.Hash(), after.Hash()),
		)

		if q.goal-before.Number.Int64() < int64(blockRequestSize) {
			continue
		}

		logger.Info("🚣 currently syncing",
			"goal", q.goal,
			"average blocks/second", q.benchmarker.mostRecentAverage(),
			"overall average", q.benchmarker.average(),
		)
	}
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

func (q *syncQueue) pushRequest(start uint64, numRequests int, to peer.ID) {
	best, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Debug("failed to get best block number", "error", err)
		return
	}

	if q.goal < best.Int64() {
		q.goal = best.Int64()
	}

	if q.goal-int64(start) < int64(blockRequestSize) {
		start := best.Int64() + 1
		req := createBlockRequest(start, 0)

		logger.Debug("pushing request to queue", "start", start)
		q.requestData.Store(start, requestData{
			received: false,
		})

		q.requestCh <- &syncRequest{
			req: req,
			to:  to,
		}
		return
	}

	// all requests must start at a multiple of 128 + 1
	m := start % uint64(blockRequestSize)
	start = start - m + 1

	for i := 0; i < numRequests; i++ {
		if start > uint64(q.goal) {
			return
		}

		req := createBlockRequest(int64(start), blockRequestSize)

		if d, has := q.requestData.Load(start); has {
			data := d.(requestData)
			// we haven't sent the request out yet, or we've already gotten the response
			if !data.sent || data.sent && data.received {
				continue
			}
		}

		logger.Debug("pushing request to queue", "start", start)

		q.requestData.Store(start, requestData{
			received: false,
		})

		q.requestCh <- &syncRequest{
			req: req,
			to:  to,
		}

		start += uint64(blockRequestSize)
	}
}

func (q *syncQueue) pushResponse(resp *BlockResponseMessage, pid peer.ID) error {
	if len(resp.BlockData) == 0 {
		return errEmptyResponseData
	}

	startHash := resp.BlockData[0].Hash
	if _, has := q.justificationRequestData.Load(startHash); has && !resp.BlockData[0].Header.Exists() {
		numJustifications := 0
		justificationResponses := []*types.BlockData{}

		for _, bd := range resp.BlockData {
			if bd.Justification.Exists() {
				justificationResponses = append(justificationResponses, bd)
				numJustifications++
			}
		}

		if numJustifications == 0 {
			return errEmptyJustificationData
		}

		q.updatePeerScore(pid, 1)
		q.justificationRequestData.Store(startHash, requestData{
			sent:     true,
			received: true,
			from:     pid,
		})

		logger.Info("pushed justification data to queue", "hash", startHash)
		q.responseCh <- justificationResponses
		return nil
	}

	start, end, err := resp.getStartAndEnd()
	if err != nil {
		// update peer's score
		q.updatePeerScore(pid, -1)
		return fmt.Errorf("response doesn't contain block headers")
	}

	if resp.BlockData[0].Body == nil || !resp.BlockData[0].Body.Exists() {
		// update peer's score
		q.updatePeerScore(pid, -1)
		return fmt.Errorf("response doesn't contain block bodies")
	}

	// update peer's score
	q.updatePeerScore(pid, 1)
	q.requestData.Store(uint64(start), requestData{
		sent:     true,
		received: true,
		from:     pid,
	})

	q.responseLock.Lock()
	defer q.responseLock.Unlock()

	for _, bd := range resp.BlockData {
		if bd.Number() == nil {
			continue
		}

		q.responses = append(q.responses, bd)
	}

	q.responses = sortResponses(q.responses)
	logger.Debug("pushed block data to queue", "start", start, "end", end, "queue", q.stringifyResponseQueue())
	return nil
}

func (q *syncQueue) processBlockRequests() {
	for {
		select {
		case req := <-q.requestCh:
			if req == nil || req.req == nil {
				continue
			}

			if !req.req.StartingBlock.IsUint64() {
				q.trySync(req)
				continue
			}

			if d, has := q.requestData.Load(req.req.StartingBlock.Uint64()); has {
				data := d.(requestData)
				if data.sent && data.received {
					continue
				}
			}

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

	logger.Trace("beginning to send out request", "start", req.req.StartingBlock.Value())
	if len(req.to) != 0 {
		resp, err := q.syncWithPeer(req.to, req.req)
		if err == nil {
			err = q.pushResponse(resp, req.to)
			if err == nil {
				return
			}
		}

		logger.Trace("failed to sync with peer", "peer", req.to, "error", err)
		q.updatePeerScore(req.to, -1)
	}

	logger.Trace("trying peers in prioritized order...")
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

		err = q.pushResponse(resp, peer.pid)
		if err != nil && err != errEmptyResponseData && err != errEmptyJustificationData {
			logger.Debug("failed to push block response", "error", err)
		} else {
			return
		}
	}

	logger.Trace("failed to sync with any peer :(")
	if req.req.StartingBlock.IsUint64() && (req.req.RequestedData&RequestedDataHeader) == 1 {
		q.requestData.Store(req.req.StartingBlock.Uint64(), requestData{
			sent:     true,
			received: false,
		})
	} else if req.req.StartingBlock.IsHash() && (req.req.RequestedData&RequestedDataHeader) == 0 {
		q.justificationRequestData.Store(req.req.StartingBlock.Hash(), requestData{
			sent:     true,
			received: false,
		})
	}

	req.to = ""
	q.requestCh <- req
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
			// if the response doesn't contain a header, then it's a justification-only response
			if !data[0].Header.Exists() {
				q.handleBlockJustification(data)
				continue
			}

			q.handleBlockData(data)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *syncQueue) handleBlockJustification(data []*types.BlockData) {
	startHash, endHash := data[0].Hash, data[len(data)-1].Hash
	logger.Debug("sending justification data to syncer", "start", startHash, "end", endHash)

	_, err := q.s.syncer.ProcessBlockData(data)
	if err != nil {
		logger.Warn("failed to handle block justifications", "error", err)
		return
	}

	logger.Debug("finished processing justification data", "start", startHash, "end", endHash)

	// update peer's score
	var from peer.ID

	d, ok := q.justificationRequestData.Load(startHash)
	if !ok {
		// this shouldn't happen
		logger.Debug("can't find request data for response!", "start", startHash)
	} else {
		from = d.(requestData).from
		q.updatePeerScore(from, 2)
		q.justificationRequestData.Delete(startHash)
	}
}

func (q *syncQueue) handleBlockData(data []*types.BlockData) {
	bestNum, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		panic(err) // TODO: don't panic but try again. seems blockState needs better concurrency handling
	}

	end := data[len(data)-1].Number().Int64()
	if end <= bestNum.Int64() {
		logger.Debug("ignoring block data that is below our head", "got", end, "head", bestNum.Int64())
		q.pushRequest(uint64(end+1), blockRequestBufferSize, "")
		return
	}

	defer func() {
		q.currStart = 0
		q.currEnd = 0
	}()

	q.currStart = data[0].Number().Int64()
	q.currEnd = end

	logger.Debug("sending block data to syncer", "start", q.currStart, "end", q.currEnd)

	idx, err := q.s.syncer.ProcessBlockData(data)
	if err != nil {
		q.handleBlockDataFailure(idx, err, data)
		return
	}

	logger.Debug("finished processing block data", "start", q.currStart, "end", q.currEnd)

	var from peer.ID
	d, ok := q.requestData.Load(uint64(q.currStart))
	if !ok {
		// this shouldn't happen
		logger.Debug("can't find request data for response!", "start", q.currStart)
	} else {
		from = d.(requestData).from
		q.updatePeerScore(from, 2)
		q.requestData.Delete(uint64(q.currStart))
	}

	q.pushRequest(uint64(q.currEnd+1), blockRequestBufferSize, from)
}

func (q *syncQueue) handleBlockDataFailure(idx int, err error, data []*types.BlockData) {
	logger.Warn("failed to handle block data", "failed on block", q.currStart+int64(idx), "error", err)

	if err.Error() == "failed to get parent hash: Key not found" { // TODO: unwrap err
		header, err := types.NewHeaderFromOptional(data[idx].Header)
		if err != nil {
			logger.Debug("failed to get header from BlockData", "idx", idx, "error", err)
			return
		}

		parentHash := header.ParentHash
		req := createBlockRequestWithHash(parentHash, 0)

		logger.Debug("pushing request for parent block", "parent", parentHash)
		q.requestCh <- &syncRequest{
			req: req,
		}
		return
	}

	q.requestData.Store(uint64(q.currStart), requestData{
		sent:     true,
		received: false,
	})
	q.pushRequest(uint64(q.currStart), 1, "")
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
	q.pushRequest(uint64(bestNum.Int64()+1), blockRequestBufferSize, from)
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

	logger.Debug("received BlockAnnounce!", "number", msg.Number, "hash", header.Hash(), "from", from)
	has, _ := q.s.blockState.HasBlockBody(header.Hash())
	if has {
		return
	}

	if header.Number.Int64() <= q.goal {
		return
	}

	bestNum, err := q.s.blockState.BestBlockNumber()
	if err != nil {
		logger.Error("failed to get best block number", "error", err)
		return
	}

	q.goal = header.Number.Int64()
	q.pushRequest(uint64(bestNum.Int64()+1), blockRequestBufferSize, from)
}

func createBlockRequest(startInt int64, size uint32) *BlockRequestMessage {
	var max *optional.Uint32
	if size != 0 {
		max = optional.NewUint32(true, size)
	} else {
		max = optional.NewUint32(false, 0)
	}

	start, _ := variadic.NewUint64OrHash(uint64(startInt))

	blockRequest := &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0, // ascending
		Max:           max,
	}

	return blockRequest
}

func createBlockRequestWithHash(startHash common.Hash, size uint32) *BlockRequestMessage {
	var max *optional.Uint32
	if size != 0 {
		max = optional.NewUint32(true, size)
	} else {
		max = optional.NewUint32(false, 0)
	}

	start, _ := variadic.NewUint64OrHash(startHash)

	blockRequest := &BlockRequestMessage{
		RequestedData: RequestedDataHeader + RequestedDataBody + RequestedDataJustification,
		StartingBlock: start,
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     0, // ascending
		Max:           max,
	}

	return blockRequest
}

func sortRequests(reqs []*syncRequest) []*syncRequest {
	if len(reqs) == 0 {
		return reqs
	}

	sort.Slice(reqs, func(i, j int) bool {
		if !reqs[i].req.StartingBlock.IsUint64() || !reqs[j].req.StartingBlock.IsUint64() {
			return false
		}

		return reqs[i].req.StartingBlock.Uint64() < reqs[j].req.StartingBlock.Uint64()
	})

	i := 0
	for {
		if i >= len(reqs)-1 {
			return reqs
		}

		if !reqs[i].req.StartingBlock.IsUint64() || !reqs[i+1].req.StartingBlock.IsUint64() {
			i++
			continue
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
