// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	// maxWorkers is the maximum number of parallel sync workers
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

var (
	bootstrapRequestData = network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification
	pendingBlocksLimit   = maxResponseSize * 32
	isSyncedGauge        = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_syncer",
		Name:      "is_synced",
		Help:      "bool representing whether the node is synced to the head of the chain",
	})
)

// peerState tracks our peers's best reported blocks
type peerState struct {
	who    peer.ID
	hash   common.Hash
	number uint
}

// workHandler handles new potential work (ie. reported peer state, block announces), results from dispatched workers,
// and stored pending work (ie. pending blocks set)
// workHandler should be implemented by `bootstrapSync` and `tipSync`
type workHandler interface {
	// handleNewPeerState returns a new worker based on a peerState.
	// The worker may be nil in which case we do nothing.
	handleNewPeerState(*peerState) (*worker, error)

	// handleWorkerResult handles the result of a worker, which may be
	// nil or error. It optionally returns a new worker to be dispatched.
	handleWorkerResult(w *worker) (workerToRetry *worker, err error)

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
	setPeerHead(p peer.ID, hash common.Hash, number uint) error

	// syncState returns the current syncing state
	syncState() chainSyncState

	// getHighestBlock returns the highest block or an error
	getHighestBlock() (highestBlock uint, err error)
}

type announcedBlock struct {
	who    peer.ID
	header *types.Header
}

type chainSync struct {
	ctx    context.Context
	cancel context.CancelFunc

	blockState BlockState
	network    Network

	workerPool      *syncWorkerPool
	blockAnnounceCh chan announcedBlock

	// tracks the latest state we know of from our peers,
	// ie. their best block hash and number
	peerStateLock sync.RWMutex
	peerState     map[peer.ID]*peerState

	// disjoint set of blocks which are known but not ready to be processed
	// ie. we only know the hash, number, or the parent block is unknown, or the body is unknown
	// note: the block may have empty fields, as some data about it may be unknown
	pendingBlocks      DisjointBlockSet
	pendingBlockDoneCh chan<- struct{}

	state       chainSyncState
	benchmarker *syncBenchmarker

	finalisedCh <-chan *types.FinalisationInfo

	minPeers     int
	slotDuration time.Duration

	logSyncTicker  *time.Ticker
	logSyncTickerC <-chan time.Time // channel as field for unit testing
	logSyncStarted bool
	logSyncDone    chan struct{}

	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry
}

type chainSyncConfig struct {
	bs                 BlockState
	net                Network
	pendingBlocks      DisjointBlockSet
	minPeers, maxPeers int
	slotDuration       time.Duration
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry
}

func newChainSync(cfg chainSyncConfig) *chainSync {
	ctx, cancel := context.WithCancel(context.Background())
	const syncSamplesToKeep = 30
	const logSyncPeriod = 3 * time.Second
	logSyncTicker := time.NewTicker(logSyncPeriod)

	return &chainSync{
		storageState:       cfg.storageState,
		transactionState:   cfg.transactionState,
		babeVerifier:       cfg.babeVerifier,
		finalityGadget:     cfg.finalityGadget,
		blockImportHandler: cfg.blockImportHandler,
		telemetry:          cfg.telemetry,
		ctx:                ctx,
		cancel:             cancel,
		blockState:         cfg.bs,
		network:            cfg.net,
		peerState:          make(map[peer.ID]*peerState),
		pendingBlocks:      cfg.pendingBlocks,
		state:              bootstrap,
		benchmarker:        newSyncBenchmarker(syncSamplesToKeep),
		finalisedCh:        cfg.bs.GetFinalisedNotifierChannel(),
		minPeers:           cfg.minPeers,
		slotDuration:       cfg.slotDuration,
		logSyncTicker:      logSyncTicker,
		logSyncTickerC:     logSyncTicker.C,
		logSyncDone:        make(chan struct{}),
		workerPool:         newSyncWorkerPool(cfg.net),
		blockAnnounceCh:    make(chan announcedBlock, cfg.maxPeers),
	}
}

func (cs *chainSync) start() {
	// wait until we have a minimal workers in the sync worker pool
	// and we have a clear target otherwise just wait
	for {
		_, err := cs.getTarget()
		totalAvailable := cs.workerPool.totalWorkers()

		if err == nil && totalAvailable >= uint(cs.minPeers) {
			break
		}

		time.Sleep(time.Millisecond * 100)
	}

	isSyncedGauge.Set(float64(cs.state))

	pendingBlockDoneCh := make(chan struct{})
	cs.pendingBlockDoneCh = pendingBlockDoneCh

	go cs.pendingBlocks.run(cs.finalisedCh, pendingBlockDoneCh)
	go cs.sync()
	cs.logSyncStarted = true
	go cs.logSyncSpeed()
}

func (cs *chainSync) stop() {
	if cs.pendingBlockDoneCh != nil {
		close(cs.pendingBlockDoneCh)
	}
	cs.cancel()
	if cs.logSyncStarted {
		<-cs.logSyncDone
	}
}

func (cs *chainSync) syncState() chainSyncState {
	return cs.state
}

func (cs *chainSync) setBlockAnnounce(who peer.ID, blockAnnounceHeader *types.Header) error {
	blockAnnounceHeaderHash := blockAnnounceHeader.Hash()
	// check if we already know of this block, if not,
	// add to pendingBlocks set
	has, err := cs.blockState.HasHeader(blockAnnounceHeaderHash)
	if err != nil {
		return err
	}

	if has {
		return blocktree.ErrBlockExists
	}

	// if the peer reports a lower or equal best block number than us,
	// check if they are on a fork or not
	bestBlockHeader, err := cs.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("best block header: %w", err)
	}

	if blockAnnounceHeader.Number <= bestBlockHeader.Number {
		// check if our block hash for that number is the same, if so, do nothing
		// as we already have that block
		ourHash, err := cs.blockState.GetHashByNumber(blockAnnounceHeader.Number)
		if err != nil {
			return fmt.Errorf("get block hash by number: %w", err)
		}

		if ourHash == blockAnnounceHeaderHash {
			return nil
		}

		// check if their best block is on an invalid chain, if it is,
		// potentially downscore them
		// for now, we can remove them from the syncing peers set
		fin, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return fmt.Errorf("get highest finalised header: %w", err)
		}

		// their block hash doesn't match ours for that number (ie. they are on a different
		// chain), and also the highest finalised block is higher than that number.
		// thus the peer is on an invalid chain
		if fin.Number >= blockAnnounceHeader.Number {
			// TODO: downscore this peer, or temporarily don't sync from them? (#1399)
			// perhaps we need another field in `peerState` to mark whether the state is valid or not
			cs.network.ReportPeer(peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			}, who)
			return fmt.Errorf("%w: for peer %s and block number %d",
				errPeerOnInvalidFork, who, blockAnnounceHeader.Number)
		}

		// peer is on a fork, check if we have processed the fork already or not
		// ie. is their block written to our db?
		has, err := cs.blockState.HasHeader(blockAnnounceHeaderHash)
		if err != nil {
			return fmt.Errorf("has header: %w", err)
		}

		// if so, do nothing, as we already have their fork
		if has {
			return nil
		}
	}

	pendingBlock := cs.pendingBlocks.getBlock(blockAnnounceHeaderHash)
	if pendingBlock != nil {
		return fmt.Errorf("block %s (#%d) in the pending set",
			blockAnnounceHeaderHash, blockAnnounceHeader.Number)
	}

	if err = cs.pendingBlocks.addHeader(blockAnnounceHeader); err != nil {
		return err
	}

	// we assume that if a peer sends us a block announce for a certain block,
	// that is also has the chain up until and including that block.
	// this may not be a valid assumption, but perhaps we can assume that
	// it is likely they will receive this block and its ancestors before us.
	cs.blockAnnounceCh <- announcedBlock{
		who:    who,
		header: blockAnnounceHeader,
	}
	return nil
}

// setPeerHead sets a peer's best known block
func (cs *chainSync) setPeerHead(who peer.ID, bestHash common.Hash, bestNumber uint) error {
	err := cs.workerPool.addWorkerFromBlockAnnounce(who)
	if err != nil {
		logger.Errorf("adding a potential worker: %s", err)
	}

	cs.peerStateLock.Lock()
	defer cs.peerStateLock.Unlock()

	cs.peerState[who] = &peerState{
		who:    who,
		hash:   bestHash,
		number: bestNumber,
	}
	return nil
}

func (cs *chainSync) logSyncSpeed() {
	defer close(cs.logSyncDone)
	defer cs.logSyncTicker.Stop()

	for {
		before, err := cs.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		if cs.state == bootstrap {
			cs.benchmarker.begin(time.Now(), before.Number)
		}

		select {
		case <-cs.logSyncTickerC: // channel of cs.logSyncTicker
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

		totalWorkers := cs.workerPool.totalWorkers()

		switch cs.state {
		case bootstrap:
			cs.benchmarker.end(time.Now(), after.Number)
			target, err := cs.getTarget()
			if errors.Is(err, errUnableToGetTarget) {
				continue
			} else if err != nil {
				logger.Errorf("while getting target: %s", err)
				continue
			}

			logger.Infof(
				"🔗 imported blocks from %d to %d (hashes [%s ... %s])",
				before.Number, after.Number, before.Hash(), after.Hash())

			logger.Infof(
				"🚣 currently syncing, %d connected peers, %d peers available to sync, "+
					"target block number %d, %.2f average blocks/second, "+
					"%.2f overall average, finalised block number %d with hash %s",
				len(cs.network.Peers()),
				totalWorkers,
				target, cs.benchmarker.mostRecentAverage(),
				cs.benchmarker.average(), finalised.Number, finalised.Hash())
		case tip:
			logger.Infof(
				"💤 node waiting, %d connected peers, %d peers available to sync, "+
					"head block number %d with hash %s, "+
					"finalised block number %d with hash %s",
				len(cs.network.Peers()),
				totalWorkers,
				after.Number, after.Hash(),
				finalised.Number, finalised.Hash())
		}
	}
}

func (cs *chainSync) sync() {
	for {
		err := cs.maybeSwitchMode()
		if err != nil {
			logger.Errorf("trying to switch mode: %w", err)
			return
		}

		if cs.state == bootstrap {
			logger.Infof("using bootstrap sync")
			err = cs.executeBootstrapSync()
		} else {
			logger.Infof("using tip sync")
			err = cs.executeTipSync()
		}

		if err != nil {
			logger.Errorf("executing bootstrap sync: %s", err)
			continue
		}
	}
}

func (cs *chainSync) executeTipSync() error {
	for {
		cs.workerPool.useConnectedPeers()
		slotDurationTimer := time.NewTimer(cs.slotDuration)

		select {
		case blockAnnouncement := <-cs.blockAnnounceCh:
			if !slotDurationTimer.Stop() {
				<-slotDurationTimer.C
			}

			who := blockAnnouncement.who
			announcedHash := blockAnnouncement.header.Hash()
			announcedNumber := blockAnnouncement.header.Number

			has, err := cs.blockState.HasHeader(announcedHash)
			if err != nil {
				return fmt.Errorf("checking if header exists: %s", err)
			}

			if has {
				continue
			}

			bestBlockHeader, err := cs.blockState.BestBlockHeader()
			if err != nil {
				return fmt.Errorf("getting best block header: %w", err)
			}

			// if the announced block contains a lower number than our best
			// block header, let's check if it is greater than our latests
			// finalized header, if so this block is likeli to be a fork
			if announcedNumber < bestBlockHeader.Number {
				highestFinalizedHeader, err := cs.blockState.GetHighestFinalisedHeader()
				if err != nil {
					return fmt.Errorf("getting highest finalized header")
				}

				// ignore the block if it has the same or lower number
				if announcedNumber <= highestFinalizedHeader.Number {
					continue
				}

				logger.Debugf("block announce lower than best block %s (#%d) and greater highest finalized %s (#%d)",
					bestBlockHeader.Hash(), bestBlockHeader.Number, highestFinalizedHeader.Hash(), highestFinalizedHeader.Number)

				parentExists, err := cs.blockState.HasHeader(blockAnnouncement.header.ParentHash)
				if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
					return fmt.Errorf("while checking header exists: %w", err)
				}

				gapLength := uint32(1)
				startAtBlock := announcedNumber
				var request *network.BlockRequestMessage

				if parentExists {
					request = singleBlockRequest(announcedHash, bootstrapRequestData)
				} else {
					gapLength = uint32(announcedNumber - highestFinalizedHeader.Number)
					startAtBlock = highestFinalizedHeader.Number + 1
					request = descendingBlockRequest(announcedHash, gapLength, bootstrapRequestData)
				}

				logger.Debugf("received a block announce from %s, requesting %d blocks, starting %s (#%d)",
					who, gapLength, announcedHash, announcedNumber)

				resultsQueue := make(chan *syncTaskResult)
				wg := sync.WaitGroup{}

				wg.Add(1)
				go cs.handleWorkersResults(resultsQueue, startAtBlock, gapLength, &wg)
				cs.workerPool.submitRequest(request, resultsQueue)
				wg.Wait()
			} else {
				gapLength := uint32(announcedNumber - bestBlockHeader.Number)
				startAtBlock := announcedNumber
				totalBlocks := uint32(1)
				var request *network.BlockRequestMessage
				if gapLength > 1 {
					request = descendingBlockRequest(announcedHash, gapLength, bootstrapRequestData)
					startAtBlock = announcedNumber - uint(*request.Max) + 1
					totalBlocks = *request.Max

					logger.Debugf("received a block announce from %s, requesting %d blocks, descending request from %s (#%d)",
						who, gapLength, announcedHash, announcedNumber)
				} else {
					gapLength = 1
					request = singleBlockRequest(announcedHash, bootstrapRequestData)
					logger.Debugf("received a block announce from %s, requesting a single block %s (#%d)",
						who, announcedHash, announcedNumber)
				}

				resultsQueue := make(chan *syncTaskResult)
				wg := sync.WaitGroup{}

				wg.Add(1)
				go cs.handleWorkersResults(resultsQueue, startAtBlock, totalBlocks, &wg)
				cs.workerPool.submitRequest(request, resultsQueue)
				wg.Wait()
			}

			err = cs.requestPendingBlocks()
			if err != nil {
				return fmt.Errorf("while requesting pending blocks")
			}
		}
	}

}

func (cs *chainSync) requestPendingBlocks() error {
	logger.Info("starting request pending blocks")
	if cs.pendingBlocks.size() == 0 {
		return nil
	}

	highestFinalized, err := cs.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("getting highest finalised header: %w", err)
	}

	for _, pendingBlock := range cs.pendingBlocks.getBlocks() {
		if pendingBlock.number <= highestFinalized.Number {
			cs.pendingBlocks.removeBlock(pendingBlock.hash)
			continue
		}

		parentExists, err := cs.blockState.HasHeader(pendingBlock.header.ParentHash)
		if err != nil {
			return fmt.Errorf("getting pending block parent header: %w", err)
		}

		if parentExists {
			err := cs.handleReadyBlock(pendingBlock.toBlockData())
			if err != nil {
				return fmt.Errorf("handling ready block: %w", err)
			}
			continue
		}

		gapLength := pendingBlock.number - highestFinalized.Number
		if gapLength > 128 {
			logger.Criticalf("GAP LENGHT: %d, GREATER THAN 128 block", gapLength)
			gapLength = 128
		}

		descendingGapRequest := descendingBlockRequest(pendingBlock.hash,
			uint32(gapLength), bootstrapRequestData)
		startAtBlock := pendingBlock.number - uint(*descendingGapRequest.Max) + 1

		// the `requests` in the tip sync are not related necessarily
		// the is why we need to treat them separately
		wg := sync.WaitGroup{}
		wg.Add(1)
		resultsQueue := make(chan *syncTaskResult)

		// TODO: we should handle the requests concurrently
		// a way of achieve that is by constructing a new `handleWorkersResults` for
		// handling only tip sync requests
		go cs.handleWorkersResults(resultsQueue, startAtBlock, *descendingGapRequest.Max, &wg)
		cs.workerPool.submitRequest(descendingGapRequest, resultsQueue)
		wg.Wait()
	}

	return nil
}

func (cs *chainSync) executeBootstrapSync() error {
	endBootstrapSync := false
	for {
		if endBootstrapSync {
			return nil
		}

		bestBlockHeader, err := cs.blockState.BestBlockHeader()
		if err != nil {
			return fmt.Errorf("getting best block header while syncing: %w", err)
		}
		startRequestAt := bestBlockHeader.Number + 1
		cs.workerPool.useConnectedPeers()

		// we build the set of requests based on the amount of available peers
		// in the worker pool, if we have more peers than `maxRequestAllowed`
		// so we limit to `maxRequestAllowed` to avoid the error
		// cannot reserve outbound connection: resource limit exceeded
		availablePeers := cs.workerPool.totalWorkers()
		if availablePeers > maxRequestAllowed {
			availablePeers = maxRequestAllowed
		}

		targetBlockNumber := startRequestAt + uint(availablePeers)*128

		realTarget, err := cs.getTarget()
		if err != nil {
			return fmt.Errorf("while getting target: %w", err)
		}

		if targetBlockNumber > realTarget {
			diff := targetBlockNumber - realTarget
			numOfRequestsToDrop := (diff / 128) + 1
			targetBlockNumber = targetBlockNumber - (numOfRequestsToDrop * 128)
			endBootstrapSync = true
		}

		requests, err := ascedingBlockRequests(
			startRequestAt, targetBlockNumber, bootstrapRequestData)
		if err != nil {
			logger.Errorf("failed to setup ascending block requests: %s", err)
		}

		expectedAmountOfBlocks := totalRequestedBlocks(requests)
		wg := sync.WaitGroup{}

		resultsQueue := make(chan *syncTaskResult)

		wg.Add(1)
		go cs.handleWorkersResults(resultsQueue, startRequestAt, expectedAmountOfBlocks, &wg)
		cs.workerPool.submitRequests(requests, resultsQueue)

		wg.Wait()
	}
}

func (cs *chainSync) maybeSwitchMode() error {
	head, err := cs.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("getting best block header: %w", err)
	}

	target, err := cs.getTarget()
	if err != nil {
		return fmt.Errorf("getting target: %w", err)
	}

	switch {
	case head.Number+maxResponseSize < target:
		// we are at least 128 blocks behind the head, switch to bootstrap
		cs.state = bootstrap
		isSyncedGauge.Set(float64(cs.state))
		logger.Debugf("switched sync mode to %d", cs.state)

	case head.Number+maxResponseSize > target:
		cs.state = tip
		isSyncedGauge.Set(float64(cs.state))
		logger.Debugf("switched sync mode to %d", cs.state)

	default:
		// head is between (target-128, target), and we don't want to switch modes.
	}

	return nil
}

var errUnableToGetTarget = errors.New("unable to get target")

// getTarget takes the average of all peer heads
// TODO: should we just return the highest? could be an attack vector potentially, if a peer reports some very large
// head block number, it would leave us in bootstrap mode forever
// it would be better to have some sort of standard deviation calculation and discard any outliers (#1861)
func (cs *chainSync) getTarget() (uint, error) {
	cs.peerStateLock.RLock()
	defer cs.peerStateLock.RUnlock()

	// in practice, this shouldn't happen, as we only start the module once we have some peer states
	if len(cs.peerState) == 0 {
		// return max uint32 instead of 0, as returning 0 would switch us to tip mode unexpectedly
		return 0, errUnableToGetTarget
	}

	// we are going to sort the data and remove the outliers then we will return the avg of all the valid elements
	uintArr := make([]uint, 0, len(cs.peerState))
	for _, ps := range cs.peerState {
		uintArr = append(uintArr, ps.number)
	}

	sum, count := nonOutliersSumCount(uintArr)
	quotientBigInt := big.NewInt(0).Div(sum, big.NewInt(int64(count)))
	return uint(quotientBigInt.Uint64()), nil
}

// handleWorkersResults, every time we submit requests to workers they results should be computed here
// and every cicle we should endup with a complete chain, whenever we identify
// any error from a worker we should evaluate the error and re-insert the request
// in the queue and wait for it to completes
func (cs *chainSync) handleWorkersResults(workersResults chan *syncTaskResult, startAtBlock uint, totalBlocks uint32, wg *sync.WaitGroup) {
	defer wg.Done()

	logger.Infof("starting handleWorkersResults, waiting %d blocks", totalBlocks)
	syncingChain := make([]*types.BlockData, totalBlocks)

loop:
	for {
		// in a case where we don't handle workers results we should check the pool
		idleDuration := time.Minute
		idleTimer := time.NewTimer(idleDuration)

		select {
		case <-idleTimer.C:
			logger.Warnf("idle ticker triggered! checking pool")
			cs.workerPool.useConnectedPeers()
			continue

		// TODO: implement a case to stop
		case taskResult := <-workersResults:
			if !idleTimer.Stop() {
				<-idleTimer.C
			}

			logger.Infof("task result: peer(%s), error: %v, hasResponse: %v",
				taskResult.who, taskResult.err != nil, taskResult.response != nil)

			if taskResult.err != nil {
				logger.Criticalf("task result error: %s", taskResult.err)

				if errors.Is(taskResult.err, network.ErrReceivedEmptyMessage) {
					cs.workerPool.submitRequest(taskResult.request, workersResults)
					continue
				}

				// TODO add this worker in a ignorePeers list, implement some expiration time for
				// peers added to it (peerJail where peers have a release date and maybe extend the punishment
				// if fail again ang again Jimmy's + Diego's idea)
				cs.workerPool.shutdownWorker(taskResult.who, true)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			}

			who := taskResult.who
			request := taskResult.request
			response := taskResult.response

			if request.Direction == network.Descending {
				// reverse blocks before pre-validating and placing in ready queue
				reverseBlockData(response.BlockData)
			}

			err := cs.validateResponse(request, response, who)
			switch {
			case errors.Is(err, errResponseIsNotChain):
				logger.Criticalf("response invalid: %s", err)
				cs.workerPool.shutdownWorker(taskResult.who, true)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			case errors.Is(err, errEmptyBlockData):
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			case errors.Is(err, errUnknownParent):
			case err != nil:
				logger.Criticalf("response invalid: %s", err)
				cs.workerPool.shutdownWorker(taskResult.who, true)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			}

			if len(response.BlockData) > 0 {
				firstBlockInResponse := response.BlockData[0]
				lastBlockInResponse := response.BlockData[len(response.BlockData)-1]

				logger.Tracef("processing %d blocks: %d (%s) to %d (%s)",
					len(response.BlockData),
					firstBlockInResponse.Header.Number, firstBlockInResponse.Hash,
					lastBlockInResponse.Header.Number, lastBlockInResponse.Hash)
			}

			for _, blockInResponse := range response.BlockData {
				blockExactIndex := blockInResponse.Header.Number - startAtBlock
				syncingChain[blockExactIndex] = blockInResponse
			}

			// we need to check if we've filled all positions
			// otherwise we should wait for more responses
			for _, element := range syncingChain {
				if element == nil {
					continue loop
				}
			}
			break loop
		}
	}

	logger.Infof("synced %d blocks, starting process", len(syncingChain))
	if len(syncingChain) >= 2 {
		// ensuring the parents are in the right place
		parentElement := syncingChain[0]
		for _, element := range syncingChain[1:] {
			if parentElement.Header.Hash() != element.Header.ParentHash {
				logger.Criticalf("expected %s be parent of %s", parentElement.Header.Hash(), element.Header.ParentHash)
				panic("")
			}

			parentElement = element
		}
	}

	// response was validated! place into ready block queue
	for _, bd := range syncingChain {
		// block is ready to be processed!
		if err := cs.handleReadyBlock(bd); err != nil {
			logger.Criticalf("error while handling a ready block: %s", err)
			return
		}
	}
}

func (cs *chainSync) handleReadyBlock(bd *types.BlockData) error {
	// if header was not requested, get it from the pending set
	// if we're expecting headers, validate should ensure we have a header
	if bd.Header == nil {
		block := cs.pendingBlocks.getBlock(bd.Hash)
		if block == nil {
			// block wasn't in the pending set!
			// let's check the db as maybe we already processed it
			has, err := cs.blockState.HasHeader(bd.Hash)
			if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
				logger.Debugf("failed to check if header is known for hash %s: %s", bd.Hash, err)
				return err
			}

			if has {
				logger.Tracef("ignoring block we've already processed, hash=%s", bd.Hash)
				return err
			}

			// this is bad and shouldn't happen
			logger.Errorf("block with unknown header is ready: hash=%s", bd.Hash)
			return err
		}

		bd.Header = block.header
	}

	if bd.Header == nil {
		logger.Errorf("new ready block number (unknown) with hash %s", bd.Hash)
		return nil
	}

	//logger.Tracef("new ready block number %d with hash %s", bd.Header.Number, bd.Hash)

	err := cs.processBlockData(*bd)
	if err != nil {
		// depending on the error, we might want to save this block for later
		logger.Errorf("block data processing for block with hash %s failed: %s", bd.Hash, err)
		return err
	}

	cs.pendingBlocks.removeBlock(bd.Hash)
	return nil
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (cs *chainSync) processBlockData(blockData types.BlockData) error { //nolint:revive
	// logger.Debugf("processing block data with hash %s", blockData.Hash)

	headerInState, err := cs.blockState.HasHeader(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has header: %w", err)
	}

	bodyInState, err := cs.blockState.HasBlockBody(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has body: %w", err)
	}

	// while in bootstrap mode we don't need to broadcast block announcements
	announceImportedBlock := cs.state == tip
	if headerInState && bodyInState {
		//logger.Infof("Process Block With State Header And Body in State: %s (#%d)", blockData.Hash.Short(), blockData.Number())
		err = cs.processBlockDataWithStateHeaderAndBody(blockData, announceImportedBlock)
		if err != nil {
			return fmt.Errorf("processing block data with header and "+
				"body in block state: %w", err)
		}
		return nil
	}

	if blockData.Header != nil {
		if blockData.Body != nil {
			//logger.Infof("Process Block With Header And Body: %s (#%d)", blockData.Hash.Short(), blockData.Number())
			err = cs.processBlockDataWithHeaderAndBody(blockData, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			logger.Infof("Process Block Justification: %s (#%d)", blockData.Hash.Short(), blockData.Number())
			err = cs.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err = cs.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (cs *chainSync) processBlockDataWithStateHeaderAndBody(blockData types.BlockData, //nolint:revive
	announceImportedBlock bool) (err error) {
	// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
	// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
	// if we update the node to only store finalised blocks in the database, this should be fixed and the entire
	// code block can be removed (#1784)
	block, err := cs.blockState.GetBlockByHash(blockData.Hash)
	if err != nil {
		return fmt.Errorf("getting block by hash: %w", err)
	}

	err = cs.blockState.AddBlockToBlockTree(block)
	if errors.Is(err, blocktree.ErrBlockExists) {
		logger.Debugf(
			"block number %d with hash %s already exists in block tree, skipping it.",
			block.Header.Number, blockData.Hash)
		return nil
	} else if err != nil {
		return fmt.Errorf("adding block to blocktree: %w", err)
	}

	if blockData.Justification != nil && len(*blockData.Justification) > 0 {
		err = cs.handleJustification(&block.Header, *blockData.Justification)
		if err != nil {
			return fmt.Errorf("handling justification: %w", err)
		}
	}

	// TODO: this is probably unnecessary, since the state is already in the database
	// however, this case shouldn't be hit often, since it's only hit if the node state
	// is rewinded or if the node shuts down unexpectedly (#1784)
	state, err := cs.storageState.TrieState(&block.Header.StateRoot)
	if err != nil {
		return fmt.Errorf("loading trie state: %w", err)
	}

	err = cs.blockImportHandler.HandleBlockImport(block, state, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block import: %w", err)
	}

	return nil
}

func (cs *chainSync) processBlockDataWithHeaderAndBody(blockData types.BlockData, //nolint:revive
	announceImportedBlock bool) (err error) {
	err = cs.babeVerifier.VerifyBlock(blockData.Header)
	if err != nil {
		return fmt.Errorf("babe verifying block: %w", err)
	}

	cs.handleBody(blockData.Body)

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = cs.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (cs *chainSync) handleBody(body *types.Body) {
	for _, ext := range *body {
		cs.transactionState.RemoveExtrinsic(ext)
	}
}

func (cs *chainSync) handleJustification(header *types.Header, justification []byte) (err error) {
	logger.Debugf("handling justification for block %d...", header.Number)

	headerHash := header.Hash()
	err = cs.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = cs.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	logger.Infof("🔨 finalised block number %d with hash %s", header.Number, headerHash)
	return nil
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (cs *chainSync) handleBlock(block *types.Block, announceImportedBlock bool) error {
	parent, err := cs.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	cs.storageState.Lock()
	defer cs.storageState.Unlock()

	ts, err := cs.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	rt, err := cs.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = cs.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	//logger.Debugf("🔗 imported block number %d with hash %s", block.Header.Number, block.Header.Hash())

	blockHash := block.Header.Hash()
	cs.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

// validateResponse performs pre-validation of a block response before placing it into either the
// pendingBlocks or readyBlocks set.
// It checks the following:
//   - the response is not empty
//   - the response contains all the expected fields
//   - each block has the correct parent, ie. the response constitutes a valid chain
func (cs *chainSync) validateResponse(req *network.BlockRequestMessage,
	resp *network.BlockResponseMessage, p peer.ID) error {
	if resp == nil || len(resp.BlockData) == 0 {
		return errEmptyBlockData
	}

	logger.Tracef("validating block response starting at block hash %s", resp.BlockData[0].Hash)

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

		if curr == nil {
			logger.Critical(">>>>>>>>>>>>>>>> CURR IS NIL!!")
		}

		// check that parent of first block in response is known (either in our db or in the ready queue)
		if i == 0 {
			prev = curr

			// check that we know the parent of the first block (or it's in the ready queue)
			has, _ := cs.blockState.HasHeader(curr.ParentHash)
			if has {
				continue
			}

			return errUnknownParent
		}

		// otherwise, check that this response forms a chain
		// ie. curr's parent hash is hash of previous header, and curr's number is previous number + 1
		if prev.Hash() != curr.ParentHash || curr.Number != prev.Number+1 {
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
		return fmt.Errorf("%w: hash=%s", errNilBodyInResponse, bd.Hash)
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

func (cs *chainSync) getHighestBlock() (highestBlock uint, err error) {
	cs.peerStateLock.RLock()
	defer cs.peerStateLock.RUnlock()

	if len(cs.peerState) == 0 {
		return 0, errNoPeers
	}

	for _, ps := range cs.peerState {
		if ps.number < highestBlock {
			continue
		}
		highestBlock = ps.number
	}

	return highestBlock, nil
}
