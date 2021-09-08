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

type (
	BlockRequestMessage  = network.BlockRequestMessage
	BlockResponseMessage = network.BlockResponseMessage
)

type chainSyncState uint64

var (
	bootstrap chainSyncState = 0
	idle      chainSyncState = 1
)

// workerState helps track the current worker set and set the upcoming worker ID
type workerState struct {
	sync.Mutex
	nextWorker uint64
	workers    map[uint64]*worker
}

func newWorkerState() *workerState {
	return &workerState{
		workers: make(map[uint64]*worker),
	}
}

func (s *workerState) add(w *worker) {
	s.Lock()
	defer s.Unlock()

	w.id = s.nextWorker
	s.nextWorker += 1
	s.workers[w.id] = w
}

func (s *workerState) delete(id uint64) {
	s.Lock()
	defer s.Unlock()
	delete(s.workers, id)
}

func (s *workerState) clear() {
	s.Lock()
	defer s.Unlock()

	for id := range s.workers {
		delete(s.workers, id)
	}
	s.nextWorker = 0
}

// worker respresents a process that is attempting to sync from the specified start block to target block
// if it fails for some reason, `err` is set.
// otherwise, we can assume all the blocks have been received and added to the `readyBlocks` queue
type worker struct {
	id uint64

	startHash    common.Hash
	startNumber  *big.Int
	targetHash   common.Hash
	targetNumber *big.Int

	// TODO: add fields to request
	direction byte

	duration time.Duration
	err      *workerError
}

type workerError struct {
	err error
	who peer.ID // whose response caused the error, if any
}

// peerState tracks our peers's best reported blocks
type peerState struct {
	who    peer.ID
	hash   common.Hash
	number *big.Int
}

// workHandler handles new potential work (ie. reported peer state, block announces), results from dispatched workers,
// and stored pending work (ie. pending blocks set)
// workHandler should be implemented by `bootstrapSync` and `idleSync`
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

	// handleTick ...
	handleTick()
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
	peerState   map[peer.ID]*peerState
	ignorePeers map[peer.ID]struct{}

	// current workers that are attempting to obtain blocks
	workerState *workerState

	// blocks which are ready to be processed are put into this channel
	// the `chainProcessor` will read from this channel and process the blocks
	// note: blocks must not be put into this channel unless their parent is known
	// TODO: channel or queue data structure?
	readyBlocks chan<- *types.BlockData

	// disjoint set of blocks which are known but not ready to be processed
	// ie. we only know the hash, number, or the parent block is unknown, or the body is unknown
	// note: the block may have empty fields, as some data about it may be unknown
	pendingBlocks DisjointBlockSet

	// bootstrap or idle (near-head)
	state chainSyncState

	// handler is set to either `bootstrapSyncer` or `idleSyncer`, depending on the current
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
	// TODO: wait until we have received ?? peer heads
	// this should be based off our min/max peers, potentially
	for {
		if len(cs.peerState) >= 1 {
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
	cs.peerState[p] = ps

	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	head, err := cs.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if ps.number.Cmp(head.Number) <= 0 {
		// check if our block hash for that number is the same, if so, do nothing
		// as we already have that block
		hash, err := cs.blockState.GetHashByNumber(ps.number)
		if err != nil {
			return err
		}

		if hash.Equal(ps.hash) {
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
			logger.Debug("peer is on an invalid fork")
			return nil
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

	cs.workQueue <- cs.peerState[p]
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

			logger.Info("🔗 imported blocks", "from", before.Number, "to", after.Number,
				"hashes", fmt.Sprintf("[%s ... %s]", before.Hash(), after.Hash()),
			)

			logger.Info("🚣 currently syncing",
				"peer count", len(cs.network.Peers()),
				// "target", target, // TODO
				"average blocks/second", cs.benchmarker.mostRecentAverage(),
				"overall average", cs.benchmarker.average(),
				"finalised", finalised.Number,
				"hash", finalised.Hash(),
			)
		case idle:
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
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case ps := <-cs.workQueue:
			// if a peer reports a greater head than us, or a chain which
			// appears to be a fork, begin syncing
			err := cs.handleWork(ps)
			if err != nil {
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
			// TODO: periodically clear out ignore list
			switch res.err.err {
			case context.DeadlineExceeded:
				if res.err.who != peer.ID("") {
					cs.ignorePeers[res.err.who] = struct{}{}
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
			cs.handler.handleTick()

			// bootstrap complete, switch state to idle if not already
			// and begin near-head fork-sync

			// TODO: create functionality to switch modes
			// will require stopping all existing workers and clearing the set
		case <-cs.ctx.Done():
			return
		}
	}
}

// handleWork handles potential new work that may be triggered on receiving a peer's state
// in bootstrap mode, this begins the bootstrap process
// in idle mode, this adds the peer's state to the pendingBlocks set and potentially starts
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
	if len(cs.workerState.workers) > MAX_WORKERS {
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

// hasCurrentWorker returns whether the current workers cover the blocks reported by this peerState
// TODO: used only by bootstrap, create targetHash version for idle?
func (cs *chainSync) hasCurrentWorker(targetNumber *big.Int) bool {
	// if we're in bootstrap mode, and there already is a worker, we don't need to dispatch another
	if cs.state == bootstrap {
		return len(cs.workerState.workers) != 0
	}

	for _, w := range cs.workerState.workers {
		if w.targetNumber.Cmp(targetNumber) >= 0 {
			// there is some worker already syncing up until this number or further
			return true
		}
	}

	return false
}

// dispatchWorker begins making requests to the network and attempts to receive responses up until the target
// if it fails due to any reason, it sets the worker `err` and returns
// this function always places the worker into the `resultCh` for result handling upon return
func (cs *chainSync) dispatchWorker(w *worker) {
	logger.Debug("dispatching sync worker", "target hash", w.targetHash, "target number", w.targetNumber)

	// to deal with descending requests (ie. target may be lower than start) which are used in idle mode,
	// take absolute value of difference between start and target
	numBlocks := int(big.NewInt(0).Abs(big.NewInt(0).Sub(w.targetNumber, w.startNumber)).Int64())
	numRequests := numBlocks / MAX_RESPONSE_SIZE

	if numBlocks < MAX_RESPONSE_SIZE {
		numRequests = 1
	}

	start := time.Now()
	defer func() {
		end := time.Now()
		w.duration = end.Sub(start)
		logger.Debug("sync worker complete", "success?", w.err == nil)
		cs.resultQueue <- w
	}()

	startNumber := w.startNumber.Uint64()

	for i := 0; i < numRequests; i++ {
		// TODO: check if we want to specify a size at any point
		// var max *optional.Uint32
		// if i == numRequests - 1 {
		// 	size := int(numBlocks) % MAX_RESPONSE_SIZE
		// 	max = optional.NewUint32(true, uint32(size))
		// } else {
		// 	max = optional.NewUint32(false, 0)
		// }

		var start *variadic.Uint64OrHash
		if w.startHash.Equal(common.EmptyHash) {
			// worker startHash is unspecified if we are in bootstrap mode
			start, _ = variadic.NewUint64OrHash(startNumber)
		} else {
			// in fork-syncing mode, we know the hash of the block on the fork we wish to sync
			start, _ = variadic.NewUint64OrHash(w.startHash)
		}

		req := &BlockRequestMessage{
			RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
			StartingBlock: start,
			// TODO: check target hash and use if fork request
			EndBlockHash: optional.NewHash(false, common.Hash{}),
			Direction:    w.direction,
			Max:          optional.NewUint32(false, 0),
		}

		// TODO: if we find a good peer, do sync with them, right now it re-selects a peer each time
		err := cs.doSync(req)
		if err != nil {
			// failed to sync, set worker error and put into result queue
			w.err = err
			return
		}

		startNumber += MAX_RESPONSE_SIZE
	}
}

func (cs *chainSync) doSync(req *BlockRequestMessage) *workerError {
	// determine which peers have the blocks we want to request
	peers := cs.determineSyncPeers(req)

	// send out request and potentially receive response, error if timeout
	var (
		resp *BlockResponseMessage
		who  peer.ID
	)

	logger.Trace("sending out block request", "request", req)

	for _, p := range peers {
		var err error
		resp, err = cs.network.DoBlockRequest(p, req)
		if err != nil {
			return &workerError{
				err: err,
				who: p,
			}
		}

		who = p
		break
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
		// TODO: if we're expecting headers, validate should ensure we have a header
		header, _ := types.NewHeaderFromOptional(bd.Header)
		logger.Trace("new ready block", "hash", bd.Hash, "number", header.Number)
		cs.readyBlocks <- bd
	}

	return nil
}

// determineSyncPeers returns a list of peers that likely have the blocks in the given block request.
// TODO: implement this
func (cs *chainSync) determineSyncPeers(_ *BlockRequestMessage) []peer.ID {
	peers := []peer.ID{}
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

	for _, bd := range resp.BlockData {
		if err := cs.validateBlockData(req, bd); err != nil {
			return err
		}
	}

	return nil
}

func (cs *chainSync) validateBlockData(req *BlockRequestMessage, bd *types.BlockData) error {
	requestedData := req.RequestedData

	if (requestedData&network.RequestedDataHeader) == 1 && bd.Header == nil {
		return errNilHeaderInResponse
	}

	return nil
}
