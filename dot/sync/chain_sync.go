package sync

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
)

const (
	// MAX_WORKERS is the maximum number of parallel sync workers
	// TODO: determine ideal value
	MAX_WORKERS = 4
)

var _ ChainSync = &chainSync{}

//nolint
type (
	BlockRequestMessage  = network.BlockRequestMessage
	BlockResponseMessage = network.BlockResponseMessage
)

type chainSyncState byte

var (
	bootstrap chainSyncState = 0 //nolint
	tip       chainSyncState = 1
)

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
	// handleWork optionally returns a new worker based on a peerState.
	// returned worker may be nil, in which case we do nothing
	handleWork(*peerState) (*worker, error)

	// handleWorkerResult handles the result of a worker, which may be
	// nil or error. optionally returns a new worker to be dispatched.
	handleWorkerResult(*worker) (*worker, error)

	// hasCurrentWorker is called before a worker is to be dispatched to
	// check whether it is a duplicate. this function returns whether there is
	// a worker that covers the scope of the proposed worker; if true,
	// ignore the proposed worker
	hasCurrentWorker(*worker, map[uint64]*worker) bool

	// handleTick handles a timer tick
	handleTick() (*worker, error)
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

	// blocks which are ready to be processed are put into this channel
	// the `chainProcessor` will read from this channel and process the blocks
	// note: blocks must not be put into this channel unless their parent is known
	// TODO: channel or queue data structure?
	// there is a case where we request and process "duplicate" blocks, which is where there
	// are some blocks in this channel, and at the same time, the bootstrap worker errors and dispatches
	// a new worker with start=(current best head), which results in the blocks in the queue
	// getting re-requested (as they have not been processed yet)
	// fix: either make this a readable queue, or track the highest block we've put into the queue
	readyBlocks chan<- *types.BlockData

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
}

func newChainSync(bs BlockState, net Network, readyBlocks chan<- *types.BlockData) *chainSync {
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
		pendingBlocks: newDisjointBlockSet(),
		state:         bootstrap,
		handler:       newBootstrapSyncer(bs),
		benchmarker:   newSyncBenchmarker(),
	}
}

func (cs *chainSync) start() {
	// wait until we have received 1+ peer heads
	// TODO: this should be based off our min/max peers
	for {
		cs.RLock()
		n := len(cs.peerState)
		cs.RUnlock()
		if n >= 1 {
			break
		}
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

	cs.pendingBlocks.addHeader(header)

	// TODO: is it ok to assume if a node announces a block that it has it + its ancestors??
	return cs.setPeerHead(from, header.Hash(), header.Number)
}

// setPeerHead sets a peer's best known block and potentially adds the peer's state to the workQueue
func (cs *chainSync) setPeerHead(p peer.ID, hash common.Hash, number *big.Int) error {
	ps := &peerState{
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
	fmt.Println(head.Number, ps.number)

	if ps.number.Cmp(head.Number) <= 0 {
		// check if our block hash for that number is the same, if so, do nothing
		// as we already have that block
		ourHash, err := cs.blockState.GetHashByNumber(ps.number)
		if err != nil {
			return err
		}

		fmt.Println(ourHash, ps.hash)

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
			// TODO: downscore this peer, or temporarily don't sync from them?
			// perhaps we need another field in `peerState` to mark whether the state is valid or not
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
	cs.pendingBlocks.addHashAndNumber(ps.hash, ps.number)

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
			// TODO: why does this function not return when ctx is cancelled???
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

		switch cs.state {
		case bootstrap:
			after, err := cs.blockState.BestBlockHeader()
			if err != nil {
				continue
			}

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
				"head", before.Number,
				"hash", before.Hash(),
				"finalised", finalised.Number,
				"hash", finalised.Hash(),
			)
		}
	}
}

func (cs *chainSync) sync() {
	// set to slot time * 2
	// TODO: make configurable
	ticker := time.NewTicker(time.Second * 12)

	for {
		select {
		case ps := <-cs.workQueue:
			head, err := cs.blockState.BestBlockHeader()
			if err != nil {
				logger.Error("failed to get best block header", "error", err)
				continue
			}

			target := cs.getTarget()
			if head.Number.Cmp(target) >= 0 {
				// bootstrap complete, switch state to tip if not already
				// and begin near-head fork-sync
				logger.Debug("switching to tip sync mode...")
				cs.switchMode(tip)
			} else if big.NewInt(0).Add(head.Number, big.NewInt(MAX_RESPONSE_SIZE)).Cmp(target) == -1 {
				// we are 128 blocks or more behind the target, switch to bootstrap mode
				logger.Debug("switching to bootstrap sync mode...")
				cs.switchMode(bootstrap)
			}

			if err := cs.handleWork(ps); err != nil {
				logger.Error("failed to handle chain sync work", "error", err)
			}
		case res := <-cs.resultQueue:
			// delete worker from workers map
			cs.workerState.delete(res.id)

			// handle results from worker
			// if there is an error, potentially retry the worker
			if res.err == nil {
				// TODO: log worker time
				continue
			}

			logger.Error("worker error", "error", res.err.err)

			// handle errors. in the case that a peer did not respond to us in time,
			// temporarily add them to the ignore list.
			// TODO: periodically clear out ignore list, currently is done if (ignore list >= peer list)
			switch res.err.err {
			case context.DeadlineExceeded:
				if res.err.who != peer.ID("") {
					cs.Lock()
					cs.ignorePeers[res.err.who] = struct{}{}
					cs.Unlock()
				}
			case context.Canceled:
				return
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
			worker, err := cs.handler.handleTick()
			if err != nil {
				logger.Error("failed to handle tick", "error", err)
				continue
			}

			if worker == nil {
				continue
			}

			cs.tryDispatchWorker(worker)
		case <-cs.ctx.Done():
			return
		}
	}
}

// switchMode stops all existing workers and clears the worker set and switches the `handler`
// based on the new mode
func (cs *chainSync) switchMode(mode chainSyncState) {
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
		cs.handler = newTipSyncer(cs.blockState, cs.pendingBlocks, cs.workerState)
	}

	cs.state = mode
}

// getTarget takes the average of all peer heads
// TODO: should we just return the highest? could be an attack vector potentially, if a peer reports some very large
// head block number, it would leave us in bootstrap mode forever
// it would be better to have some sort of standard deviation calculation and discard any outliers
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
	worker, err := cs.handler.handleWork(ps)
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
	if len(cs.workerState.workers) >= MAX_WORKERS {
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
		"target hash", w.targetHash,
		"target number", w.targetNumber,
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
		logger.Debug("sync worker complete", "success?", w.err == nil)
		cs.resultQueue <- w
	}()

	reqs, err := workerToRequests(w)
	if err != nil {
		// if we are creating valid workers, this should not happen
		logger.Crit("failed to create requests from worker", "worker", w, "error", err)
	}

	for _, req := range reqs {
		// TODO: if we find a good peer, do sync with them, right now it re-selects a peer each time
		err := cs.doSync(req)
		if err != nil {
			// failed to sync, set worker error and put into result queue
			w.err = err
			return
		}
	}
}

func workerToRequests(w *worker) ([]*BlockRequestMessage, error) {
	// one of start number or hash must be provided
	if w.startNumber == nil && w.startHash.Equal(common.EmptyHash) {
		return nil, errWorkerMissingStartBlock
	}

	// worker must specify a target number
	// empty target hash is ok (eg. in the case of descending fork requests)
	if w.targetNumber == nil {
		return nil, errWorkerMissingTargetNumber
	}

	// to deal with descending requests (ie. target may be lower than start) which are used in tip mode,
	// take absolute value of difference between start and target
	numBlocks := int(big.NewInt(0).Abs(big.NewInt(0).Sub(w.targetNumber, w.startNumber)).Int64())
	numRequests := numBlocks / MAX_RESPONSE_SIZE

	if numBlocks < MAX_RESPONSE_SIZE {
		numRequests = 1
	}

	startNumber := w.startNumber.Uint64()

	reqs := make([]*BlockRequestMessage, numRequests)

	for i := 0; i < numRequests; i++ {
		// check if we want to specify a size
		var max *optional.Uint32
		if i == numRequests-1 {
			size := numBlocks % MAX_RESPONSE_SIZE
			if size == 0 {
				size = MAX_RESPONSE_SIZE
			}
			max = optional.NewUint32(true, uint32(size))
		} else {
			max = optional.NewUint32(false, 0)
		}

		var start *variadic.Uint64OrHash
		if w.startHash.Equal(common.EmptyHash) {
			// worker startHash is unspecified if we are in bootstrap mode
			start, _ = variadic.NewUint64OrHash(startNumber)
		} else {
			// in tip-syncing mode, we know the hash of the block on the fork we wish to sync
			start, _ = variadic.NewUint64OrHash(w.startHash)
		}

		reqs[i] = &BlockRequestMessage{
			RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
			StartingBlock: start,
			// TODO: check target hash and use if fork request
			EndBlockHash: optional.NewHash(false, common.Hash{}),
			Direction:    w.direction,
			Max:          max,
		}
		startNumber += MAX_RESPONSE_SIZE
	}

	return reqs, nil
}

func (cs *chainSync) doSync(req *BlockRequestMessage) *workerError {
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

	// TODO: either randomly sort or use scoring to determine what peer to try to sync from
	who := peers[0]
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

	if req.Direction == DIR_DESCENDING {
		// reverse blocks before pre-validating and placing in ready queue
		tmp := make([]*types.BlockData, len(resp.BlockData))
		for i, bd := range resp.BlockData {
			tmp[len(tmp)-1-i] = bd
		}
		resp.BlockData = tmp
	}

	// perform some pre-validation of response, error if failure
	if err := cs.validateResponse(req, resp); err != nil {
		return &workerError{
			err: err,
			who: who,
		}
	}

	logger.Trace("success! placing block response data in ready queue")

	// response was validated! place into ready block queue
	for _, bd := range resp.BlockData {
		// if we're expecting headers, validate should ensure we have a header
		header, _ := types.NewHeaderFromOptional(bd.Header)

		// block is ready to be processed!
		logger.Trace("new ready block", "hash", bd.Hash, "number", header.Number)
		cs.pendingBlocks.removeBlock(bd.Hash)
		cs.readyBlocks <- bd
	}

	return nil
}

// determineSyncPeers returns a list of peers that likely have the blocks in the given block request.
// TODO: implement this
func (cs *chainSync) determineSyncPeers(_ *BlockRequestMessage) []peer.ID {
	peers := []peer.ID{}

	cs.RLock()
	defer cs.RUnlock()

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
func (cs *chainSync) validateResponse(req *BlockRequestMessage, resp *BlockResponseMessage) error {
	if len(resp.BlockData) == 0 {
		return errEmptyBlockData
	}

	logger.Trace("validating block response", "start", resp.BlockData[0].Hash)

	var (
		prev, curr *types.Header
		err        error
	)
	headerRequested := (req.RequestedData & network.RequestedDataHeader) == 1

	for i, bd := range resp.BlockData {
		if err = cs.validateBlockData(req, bd); err != nil {
			return err
		}

		if headerRequested {
			curr, err = types.NewHeaderFromOptional(bd.Header)
			if err != nil {
				return err
			}
		} else {
			// TODO: if this is a justification-only request, make sure we have the block for the justification
			continue
		}

		// check that parent of first block in response is known (either in our db or in the ready queue)
		if i == 0 {
			// TODO
			prev = curr
			continue
		}

		// otherwise, check that this response forms a chain
		if !prev.Hash().Equal(curr.ParentHash) {
			// the response is missing some blocks, place blocks from curr onwards into pending blocks set
			for _, bd := range resp.BlockData[i:] {
				body, err := types.NewBodyFromOptional(bd.Body)
				if err != nil {
					return fmt.Errorf("failed to convert block body from optional: hash=%s err=%s", bd.Hash, err)
				}

				cs.pendingBlocks.addBlock(&types.Block{
					Header: curr,
					Body:   body,
				})
			}
			return errResponseIsNotChain
		}

		prev = curr
	}

	return nil
}

// validateBlockData checks that the expected fields are in the block data
func (cs *chainSync) validateBlockData(req *BlockRequestMessage, bd *types.BlockData) error {
	if bd == nil {
		return errNilBlockData
	}

	requestedData := req.RequestedData

	if (requestedData&network.RequestedDataHeader) == 1 && bd.Header == nil {
		return errNilHeaderInResponse
	}

	if (requestedData&network.RequestedDataBody) == 1 && bd.Body == nil {
		return errNilBodyInResponse
	}

	return nil
}
