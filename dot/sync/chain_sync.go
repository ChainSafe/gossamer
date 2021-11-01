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

package sync

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
)

const (
	// maxWorkers is the maximum number of parallel sync workers
	// TODO: determine ideal value (#1659)
	maxWorkers = 12
)

var _ ChainSync = &chainSync{}

type chainSyncState byte

const (
	bootstrap chainSyncState = iota
	tip
)

func (s chainSyncState) String() string {
	switch s {
	case bootstrap:
		return "bootstrap"
	case tip:
		return "tip"
	default:
		return "unknown"
	}
}

// TODO: determine ideal limit for pending blocks set (#1659)
var pendingBlocksLimit = maxResponseSize * 32

// peerState tracks our peers's best reported blocks
type peerState struct {
	who    peer.ID //nolint
	hash   common.Hash
	number *big.Int
}

// workHandler handles new potential work (ie. reported peer state, block announces), results from dispatched workers,
// and stored pending work (ie. pending blocks set)
// workHandler should be implemented by `bootstrapSync` and `tipSync`
type workHandler interface {
	// handleNewPeerState optionally returns a new worker based on a peerState.
	// returned worker may be nil, in which case we do nothing
	handleNewPeerState(*peerState) (*worker, error)

	// handleWorkerResult handles the result of a worker, which may be
	// nil or error. optionally returns a new worker to be dispatched.
	handleWorkerResult(*worker) (*worker, error)

	// hasCurrentWorker is called before a worker is to be dispatched to
	// check whether it is a duplicate. this function returns whether there is
	// a worker that covers the scope of the proposed worker; if true,
	// ignore the proposed worker
	hasCurrentWorker(*worker, map[uint64]*worker) bool

	// handleTick handles a timer tick
	handleTick() ([]*worker, error)
}

// ChainSync contains the methods used by the high-level service into the `chainSync` module
type ChainSync interface {
	start()
	stop()

	// called upon receiving a BlockAnnounce
	setBlockAnnounce(from peer.ID, header *types.Header) error

	// called upon receiving a BlockAnnounceHandshake
	setPeerHead(p peer.ID, hash common.Hash, number *big.Int) error

	// syncState returns the current syncing state
	syncState() chainSyncState
}

type chainSync struct {
	ctx    context.Context
	cancel context.CancelFunc

	blockState BlockState
	network    Network

	// queue of work created by setting peer heads
	workQueue chan *peerState

	// workers are put here when they are completed so we can handle their result
	resultQueue chan *worker

	// tracks the latest state we know of from our peers,
	// ie. their best block hash and number
	sync.RWMutex
	peerState   map[peer.ID]*peerState
	ignorePeers map[peer.ID]struct{}

	// current workers that are attempting to obtain blocks
	workerState *workerState

	// blocks which are ready to be processed are put into this queue
	// the `chainProcessor` will read from this channel and process the blocks
	// note: blocks must not be put into this channel unless their parent is known
	//
	// there is a case where we request and process "duplicate" blocks, which is where there
	// are some blocks in this queue, and at the same time, the bootstrap worker errors and dispatches
	// a new worker with start=(current best head), which results in the blocks in the queue
	// getting re-requested (as they have not been processed yet)
	// to fix this, we track the blocks that are in the queue
	readyBlocks *blockQueue

	// disjoint set of blocks which are known but not ready to be processed
	// ie. we only know the hash, number, or the parent block is unknown, or the body is unknown
	// note: the block may have empty fields, as some data about it may be unknown
	pendingBlocks DisjointBlockSet

	// bootstrap or tip (near-head)
	state chainSyncState

	// handler is set to either `bootstrapSyncer` or `tipSyncer`, depending on the current
	// chain sync state
	handler workHandler

	benchmarker *syncBenchmarker

	finalisedCh <-chan *types.FinalisationInfo

	minPeers     int
	slotDuration time.Duration
}

func newChainSync(bs BlockState, net Network, readyBlocks *blockQueue, pendingBlocks DisjointBlockSet, minPeers int, slotDuration time.Duration) *chainSync {
	ctx, cancel := context.WithCancel(context.Background())
	return &chainSync{
		ctx:           ctx,
		cancel:        cancel,
		blockState:    bs,
		network:       net,
		workQueue:     make(chan *peerState, 1024),
		resultQueue:   make(chan *worker, 1024),
		peerState:     make(map[peer.ID]*peerState),
		ignorePeers:   make(map[peer.ID]struct{}),
		workerState:   newWorkerState(),
		readyBlocks:   readyBlocks,
		pendingBlocks: pendingBlocks,
		state:         bootstrap,
		handler:       newBootstrapSyncer(bs),
		benchmarker:   newSyncBenchmarker(),
		finalisedCh:   bs.GetFinalisedNotifierChannel(),
		minPeers:      minPeers,
		slotDuration:  slotDuration,
	}
}

func (cs *chainSync) start() {
	// wait until we have received at least `minPeers` peer heads
	for {
		cs.RLock()
		n := len(cs.peerState)
		cs.RUnlock()
		if n >= cs.minPeers {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	go cs.sync()
	go cs.logSyncSpeed()
}

func (cs *chainSync) stop() {
	cs.cancel()
}

func (cs *chainSync) syncState() chainSyncState {
	return cs.state
}

func (cs *chainSync) setBlockAnnounce(from peer.ID, header *types.Header) error {
	// check if we already know of this block, if not,
	// add to pendingBlocks set
	has, err := cs.blockState.HasHeader(header.Hash())
	if err != nil {
		return err
	}

	if has {
		return nil
	}

	if err = cs.pendingBlocks.addHeader(header); err != nil {
		return err
	}

	// TODO: is it ok to assume if a node announces a block that it has it + its ancestors? (#1659)
	return cs.setPeerHead(from, header.Hash(), header.Number)
}

// setPeerHead sets a peer's best known block and potentially adds the peer's state to the workQueue
func (cs *chainSync) setPeerHead(p peer.ID, hash common.Hash, number *big.Int) error {
	ps := &peerState{
		who:    p,
		hash:   hash,
		number: number,
	}
	cs.Lock()
	cs.peerState[p] = ps
	cs.Unlock()

	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	head, err := cs.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if ps.number.Cmp(head.Number) <= 0 {
		// check if our block hash for that number is the same, if so, do nothing
		// as we already have that block
		ourHash, err := cs.blockState.GetHashByNumber(ps.number) //nolint
		if err != nil {
			return err
		}

		if ourHash.Equal(ps.hash) {
			return nil
		}

		// check if their best block is on an invalid chain, if it is,
		// potentially downscore them
		// for now, we can remove them from the syncing peers set
		fin, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return err
		}

		// their block hash doesn't match ours for that number (ie. they are on a different
		// chain), and also the highest finalised block is higher than that number.
		// thus the peer is on an invalid chain
		if fin.Number.Cmp(ps.number) >= 0 {
			// TODO: downscore this peer, or temporarily don't sync from them? (#1399)
			// perhaps we need another field in `peerState` to mark whether the state is valid or not
			cs.network.ReportPeer(peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			}, p)
			return errPeerOnInvalidFork
		}

		// peer is on a fork, check if we have processed the fork already or not
		// ie. is their block written to our db?
		has, err := cs.blockState.HasHeader(ps.hash)
		if err != nil {
			return err
		}

		// if so, do nothing, as we already have their fork
		if has {
			return nil
		}
	}

	// the peer has a higher best block than us, or they are on some fork we are not aware of
	// add it to the disjoint block set
	if err = cs.pendingBlocks.addHashAndNumber(ps.hash, ps.number); err != nil {
		return err
	}

	cs.workQueue <- ps
	logger.Debug("set peer head", "peer", p, "hash", hash, "number", number)
	return nil
}

func (cs *chainSync) logSyncSpeed() {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	for {
		before, err := cs.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		if cs.state == bootstrap {
			cs.benchmarker.begin(before.Number.Uint64())
		}

		select {
		case <-t.C:
			if cs.ctx.Err() != nil {
				return
			}
		case <-cs.ctx.Done():
			return
		}

		finalised, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			continue
		}

		after, err := cs.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		switch cs.state {
		case bootstrap:
			cs.benchmarker.end(after.Number.Uint64())
			target := cs.getTarget()

			logger.Info("🔗 imported blocks", "from", before.Number, "to", after.Number,
				"hashes", fmt.Sprintf("[%s ... %s]", before.Hash(), after.Hash()),
			)

			logger.Info("🚣 currently syncing",
				"peer count", len(cs.network.Peers()),
				"target", target,
				"average blocks/second", cs.benchmarker.mostRecentAverage(),
				"overall average", cs.benchmarker.average(),
				"finalised", finalised.Number,
				"hash", finalised.Hash(),
			)
		case tip:
			logger.Info("💤 node waiting",
				"peer count", len(cs.network.Peers()),
				"head", after.Number,
				"hash", after.Hash(),
				"finalised", finalised.Number,
				"hash", finalised.Hash(),
			)
		}
	}
}

func (cs *chainSync) ignorePeer(who peer.ID) {
	if err := who.Validate(); err != nil {
		return
	}

	cs.Lock()
	cs.ignorePeers[who] = struct{}{}
	cs.Unlock()
}

func (cs *chainSync) sync() {
	// set to slot time
	ticker := time.NewTicker(cs.slotDuration)

	for {
		select {
		case ps := <-cs.workQueue:
			cs.maybeSwitchMode()

			if err := cs.handleWork(ps); err != nil {
				logger.Error("failed to handle chain sync work", "error", err)
			}
		case res := <-cs.resultQueue:
			// delete worker from workers map
			cs.workerState.delete(res.id)

			// handle results from worker
			// if there is an error, potentially retry the worker
			if res.err == nil || res.ctx.Err() != nil {
				continue
			}

			logger.Debug("worker error", "error", res.err.err)

			// handle errors. in the case that a peer did not respond to us in time,
			// temporarily add them to the ignore list.
			switch {
			case errors.Is(res.err.err, context.Canceled):
				return
			case errors.Is(res.err.err, context.DeadlineExceeded):
				cs.network.ReportPeer(peerset.ReputationChange{
					Value:  peerset.TimeOutValue,
					Reason: peerset.TimeOutReason,
				}, res.err.who)
				cs.ignorePeer(res.err.who)
			case strings.Contains(res.err.err.Error(), "dial backoff"):
				cs.ignorePeer(res.err.who)
				continue
			case res.err.err.Error() == "protocol not supported":
				cs.network.ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadProtocolValue,
					Reason: peerset.BadProtocolReason,
				}, res.err.who)
				cs.ignorePeer(res.err.who)
				continue
			default:
			}

			worker, err := cs.handler.handleWorkerResult(res)
			if err != nil {
				logger.Error("failed to handle worker result", "error", err)
				continue
			}

			if worker == nil {
				continue
			}

			cs.tryDispatchWorker(worker)
		case <-ticker.C:
			cs.maybeSwitchMode()

			workers, err := cs.handler.handleTick()
			if err != nil {
				logger.Error("failed to handle tick", "error", err)
				continue
			}

			for _, worker := range workers {
				cs.tryDispatchWorker(worker)
			}
		case fin := <-cs.finalisedCh:
			// on finalised block, call pendingBlocks.removeLowerBlocks() to remove blocks on
			// invalid forks from the pending blocks set
			cs.pendingBlocks.removeLowerBlocks(fin.Header.Number)
		case <-cs.ctx.Done():
			return
		}
	}
}

func (cs *chainSync) maybeSwitchMode() {
	head, err := cs.blockState.BestBlockHeader()
	if err != nil {
		logger.Error("failed to get best block header", "error", err)
		return
	}

	target := cs.getTarget()
	switch {
	case big.NewInt(0).Add(head.Number, big.NewInt(maxResponseSize)).Cmp(target) < 0:
		// we are at least 128 blocks behind the head, switch to bootstrap
		cs.setMode(bootstrap)
	case head.Number.Cmp(target) >= 0:
		// bootstrap complete, switch state to tip if not already
		// and begin near-head fork-sync
		cs.setMode(tip)
	default:
		// head is between (target-128, target), and we don't want to switch modes.
	}
}

// setMode stops all existing workers and clears the worker set and switches the `handler`
// based on the new mode, if the mode is different than previous
func (cs *chainSync) setMode(mode chainSyncState) {
	if cs.state == mode {
		return
	}

	// stop all current workers and clear set
	cs.workerState.reset()

	// update handler to respective mode
	switch mode {
	case bootstrap:
		cs.handler = newBootstrapSyncer(cs.blockState)
	case tip:
		cs.handler = newTipSyncer(cs.blockState, cs.pendingBlocks, cs.readyBlocks)
	}

	cs.state = mode
	logger.Debug("switched sync mode", "mode", mode)
}

// getTarget takes the average of all peer heads
// TODO: should we just return the highest? could be an attack vector potentially, if a peer reports some very large
// head block number, it would leave us in bootstrap mode forever
// it would be better to have some sort of standard deviation calculation and discard any outliers (#1861)
func (cs *chainSync) getTarget() *big.Int {
	count := int64(0)
	sum := big.NewInt(0)

	cs.RLock()
	defer cs.RUnlock()

	// in practice, this shouldn't happen, as we only start the module once we have some peer states
	if len(cs.peerState) == 0 {
		// return max uint32 instead of 0, as returning 0 would switch us to tip mode unexpectedly
		return big.NewInt(2<<32 - 1)
	}

	for _, ps := range cs.peerState {
		sum = big.NewInt(0).Add(sum, ps.number)
		count++
	}

	return big.NewInt(0).Div(sum, big.NewInt(count))
}

// handleWork handles potential new work that may be triggered on receiving a peer's state
// in bootstrap mode, this begins the bootstrap process
// in tip mode, this adds the peer's state to the pendingBlocks set and potentially starts
// a fork sync
func (cs *chainSync) handleWork(ps *peerState) error {
	logger.Trace("handling potential work", "target hash", ps.hash, "target number", ps.number)
	worker, err := cs.handler.handleNewPeerState(ps)
	if err != nil {
		return err
	}

	if worker == nil {
		return nil
	}

	cs.tryDispatchWorker(worker)
	return nil
}

func (cs *chainSync) tryDispatchWorker(w *worker) {
	// if we already have the maximum number of workers, don't dispatch another
	if len(cs.workerState.workers) >= maxWorkers {
		logger.Trace("reached max workers, ignoring potential work")
		return
	}

	// check current worker set for workers already working on these blocks
	// if there are none, dispatch new worker
	if cs.handler.hasCurrentWorker(w, cs.workerState.workers) {
		return
	}

	cs.workerState.add(w)
	go cs.dispatchWorker(w)
}

// dispatchWorker begins making requests to the network and attempts to receive responses up until the target
// if it fails due to any reason, it sets the worker `err` and returns
// this function always places the worker into the `resultCh` for result handling upon return
func (cs *chainSync) dispatchWorker(w *worker) {
	logger.Debug("dispatching sync worker",
		"id", w.id,
		"start number", w.startNumber,
		"start hash", w.startHash,
		"target hash", w.targetHash,
		"target number", w.targetNumber,
		"request data", w.requestData,
		"direction", w.direction,
	)

	if w.targetNumber == nil || w.startNumber == nil {
		logger.Error("must provide a block start and target number",
			"startNumber==nil?", w.startNumber == nil,
			"targetNumber==nil?", w.targetNumber == nil,
		)
		return
	}

	start := time.Now()
	defer func() {
		end := time.Now()
		w.duration = end.Sub(start)
		logger.Debug("sync worker complete",
			"id", w.id,
			"success?", w.err == nil,
			"duration", w.duration,
		)
		cs.resultQueue <- w
	}()

	reqs, err := workerToRequests(w)
	if err != nil {
		// if we are creating valid workers, this should not happen
		logger.Crit("failed to create requests from worker", "worker", w, "error", err)
		w.err = &workerError{
			err: err,
		}
		return
	}

	for _, req := range reqs {
		// TODO: if we find a good peer, do sync with them, right now it re-selects a peer each time (#1399)
		if err := cs.doSync(req); err != nil {
			// failed to sync, set worker error and put into result queue
			w.err = err
			return
		}
	}
}

func (cs *chainSync) doSync(req *network.BlockRequestMessage) *workerError {
	// determine which peers have the blocks we want to request
	peers := cs.determineSyncPeers(req)

	if len(peers) == 0 {
		cs.Lock()
		for p := range cs.ignorePeers {
			delete(cs.ignorePeers, p)
		}

		for p := range cs.peerState {
			peers = append(peers, p)
		}
		cs.Unlock()
	}

	if len(peers) == 0 {
		return &workerError{
			err: errNoPeers,
		}
	}

	// send out request and potentially receive response, error if timeout
	logger.Trace("sending out block request", "request", req)

	// TODO: use scoring to determine what peer to try to sync from first (#1399)
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(peers))))
	who := peers[idx.Int64()]
	resp, err := cs.network.DoBlockRequest(who, req)
	if err != nil {
		return &workerError{
			err: err,
			who: who,
		}
	}

	if resp == nil {
		return &workerError{
			err: errNilResponse,
			who: who,
		}
	}

	if req.Direction == network.Descending {
		// reverse blocks before pre-validating and placing in ready queue
		reverseBlockData(resp.BlockData)
	}

	// perform some pre-validation of response, error if failure
	if err := cs.validateResponse(req, resp, who); err != nil {
		return &workerError{
			err: err,
			who: who,
		}
	}

	logger.Trace("success! placing block response data in ready queue")

	// response was validated! place into ready block queue
	for _, bd := range resp.BlockData {
		// block is ready to be processed!
		handleReadyBlock(bd, cs.pendingBlocks, cs.readyBlocks)
	}

	return nil
}

func handleReadyBlock(bd *types.BlockData, pendingBlocks DisjointBlockSet, readyBlocks *blockQueue) {
	// see if there are any descendents in the pending queue that are now ready to be processed,
	// as we have just become aware of their parent block

	// if header was not requested, get it from the pending set
	// if we're expecting headers, validate should ensure we have a header
	if bd.Header == nil {
		block := pendingBlocks.getBlock(bd.Hash)
		bd.Header = block.header
	}

	logger.Trace("new ready block", "hash", bd.Hash, "number", bd.Header.Number)

	ready := []*types.BlockData{bd}
	ready = pendingBlocks.getReadyDescendants(bd.Hash, ready)

	for _, rb := range ready {
		pendingBlocks.removeBlock(rb.Hash)
		readyBlocks.push(rb)
	}
}

// determineSyncPeers returns a list of peers that likely have the blocks in the given block request.
func (cs *chainSync) determineSyncPeers(_ *network.BlockRequestMessage) []peer.ID {
	cs.RLock()
	defer cs.RUnlock()

	peers := make([]peer.ID, 0, len(cs.peerState))

	for p := range cs.peerState {
		if _, has := cs.ignorePeers[p]; has {
			continue
		}

		peers = append(peers, p)
	}

	return peers
}

// validateResponse performs pre-validation of a block response before placing it into either the
// pendingBlocks or readyBlocks set.
// It checks the following:
// 	- the response is not empty
//  - the response contains all the expected fields
//  - each block has the correct parent, ie. the response constitutes a valid chain
func (cs *chainSync) validateResponse(req *network.BlockRequestMessage, resp *network.BlockResponseMessage, p peer.ID) error {
	if resp == nil || len(resp.BlockData) == 0 {
		return errEmptyBlockData
	}

	logger.Trace("validating block response", "start", resp.BlockData[0].Hash)

	var (
		prev, curr *types.Header
		err        error
	)
	headerRequested := (req.RequestedData & network.RequestedDataHeader) == 1

	for i, bd := range resp.BlockData {
		if err = cs.validateBlockData(req, bd, p); err != nil {
			return err
		}

		if headerRequested {
			curr = bd.Header
		} else {
			// if this is a justification-only request, make sure we have the block for the justification
			if err = cs.validateJustification(bd); err != nil {
				cs.network.ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadJustificationValue,
					Reason: peerset.BadJustificationReason,
				}, p)
				return err
			}
			continue
		}

		// check that parent of first block in response is known (either in our db or in the ready queue)
		if i == 0 {
			prev = curr

			// check that we know the parent of the first block (or it's in the ready queue)
			has, _ := cs.blockState.HasHeader(curr.ParentHash)
			if has {
				continue
			}

			if cs.readyBlocks.has(curr.ParentHash) {
				continue
			}

			// parent unknown, add to pending blocks
			if err := cs.pendingBlocks.addBlock(&types.Block{
				Header: *curr,
				Body:   *bd.Body,
			}); err != nil {
				return err
			}

			if bd.Justification != nil {
				if err := cs.pendingBlocks.addJustification(bd.Hash, *bd.Justification); err != nil {
					return err
				}
			}

			return errUnknownParent
		}

		// otherwise, check that this response forms a chain
		// ie. curr's parent hash is hash of previous header, and curr's number is previous number + 1
		if !prev.Hash().Equal(curr.ParentHash) || curr.Number.Cmp(big.NewInt(0).Add(prev.Number, big.NewInt(1))) != 0 {
			// the response is missing some blocks, place blocks from curr onwards into pending blocks set
			for _, bd := range resp.BlockData[i:] {
				if err := cs.pendingBlocks.addBlock(&types.Block{
					Header: *curr,
					Body:   *bd.Body,
				}); err != nil {
					return err
				}

				if bd.Justification != nil {
					if err := cs.pendingBlocks.addJustification(bd.Hash, *bd.Justification); err != nil {
						return err
					}
				}
			}
			return errResponseIsNotChain
		}

		prev = curr
	}

	return nil
}

// validateBlockData checks that the expected fields are in the block data
func (cs *chainSync) validateBlockData(req *network.BlockRequestMessage, bd *types.BlockData, p peer.ID) error {
	if bd == nil {
		return errNilBlockData
	}

	requestedData := req.RequestedData

	if (requestedData&network.RequestedDataHeader) == 1 && bd.Header == nil {
		cs.network.ReportPeer(peerset.ReputationChange{
			Value:  peerset.IncompleteHeaderValue,
			Reason: peerset.IncompleteHeaderReason,
		}, p)
		return errNilHeaderInResponse
	}

	if (requestedData&network.RequestedDataBody>>1) == 1 && bd.Body == nil {
		return errNilBodyInResponse
	}

	return nil
}

func (cs *chainSync) validateJustification(bd *types.BlockData) error {
	if bd == nil {
		return errNilBlockData
	}

	// this is ok, since the remote peer doesn't need to provide the info we request from them
	// especially with justifications, it's common that they don't have them.
	if bd.Justification == nil {
		return nil
	}

	has, _ := cs.blockState.HasHeader(bd.Hash)
	if !has {
		return errUnknownBlockForJustification
	}

	return nil
}

func workerToRequests(w *worker) ([]*network.BlockRequestMessage, error) {
	// worker must specify a start number
	// empty start hash is ok (eg. in the case of bootstrap, start hash is unknown)
	if w.startNumber == nil {
		return nil, errWorkerMissingStartNumber
	}

	// worker must specify a target number
	// empty target hash is ok (eg. in the case of descending fork requests)
	if w.targetNumber == nil {
		return nil, errWorkerMissingTargetNumber
	}

	diff := big.NewInt(0).Sub(w.targetNumber, w.startNumber)
	if diff.Int64() < 0 && w.direction != network.Descending {
		return nil, errInvalidDirection
	}

	if diff.Int64() > 0 && w.direction != network.Ascending {
		return nil, errInvalidDirection
	}

	// start and end block are the same, just request 1 block
	if diff.Cmp(big.NewInt(0)) == 0 {
		diff = big.NewInt(1)
	}

	// to deal with descending requests (ie. target may be lower than start) which are used in tip mode,
	// take absolute value of difference between start and target
	numBlocks := int(big.NewInt(0).Abs(diff).Int64())
	numRequests := numBlocks / maxResponseSize

	if numBlocks%maxResponseSize != 0 {
		numRequests++
	}

	startNumber := w.startNumber.Uint64()
	reqs := make([]*network.BlockRequestMessage, numRequests)

	for i := 0; i < numRequests; i++ {
		// check if we want to specify a size
		var max uint32 = maxResponseSize
		if i == numRequests-1 {
			size := numBlocks % maxResponseSize
			if size == 0 {
				size = maxResponseSize
			}
			max = uint32(size)
		}

		var start *variadic.Uint64OrHash
		if w.startHash.IsEmpty() {
			// worker startHash is unspecified if we are in bootstrap mode
			start, _ = variadic.NewUint64OrHash(startNumber)
		} else {
			// in tip-syncing mode, we know the hash of the block on the fork we wish to sync
			start, _ = variadic.NewUint64OrHash(w.startHash)

			// if we're doing descending requests and not at the last (highest starting) request,
			// then use number as start block
			if w.direction == network.Descending && i != numRequests-1 {
				start = variadic.MustNewUint64OrHash(startNumber)
			}
		}

		var end *common.Hash
		if !w.targetHash.IsEmpty() && i == numRequests-1 {
			// if we're on our last request (which should contain the target hash),
			// then add it
			end = &w.targetHash
		}

		reqs[i] = &network.BlockRequestMessage{
			RequestedData: w.requestData,
			StartingBlock: *start,
			EndBlockHash:  end,
			Direction:     w.direction,
			Max:           &max,
		}

		switch w.direction {
		case network.Ascending:
			startNumber += maxResponseSize
		case network.Descending:
			startNumber -= maxResponseSize
		}
	}

	// if our direction is descending, we want to send out the request with the lowest
	// startNumber first
	if w.direction == network.Descending {
		for i, j := 0, len(reqs)-1; i < j; i, j = i+1, j-1 {
			reqs[i], reqs[j] = reqs[j], reqs[i]
		}
	}

	return reqs, nil
}
