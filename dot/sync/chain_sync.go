// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
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

// peerView tracks our peers's best reported blocks
type peerView struct {
	who    peer.ID
	hash   common.Hash
	number uint
}

// ChainSync contains the methods used by the high-level service into the `chainSync` module
type ChainSync interface {
	start()
	stop()

	// called upon receiving a BlockAnnounceHandshake
	setPeerHead(p peer.ID, hash common.Hash, number uint)

	// syncState returns the current syncing state
	syncState() chainSyncState

	// getHighestBlock returns the highest block or an error
	getHighestBlock() (highestBlock uint, err error)

	onImportBlock(announcedBlock) error
}

type announcedBlock struct {
	who    peer.ID
	header *types.Header
}

type chainSync struct {
	stopCh chan struct{}

	blockState BlockState
	network    Network

	workerPool *syncWorkerPool

	// tracks the latest state we know of from our peers,
	// ie. their best block hash and number
	peerViewLock sync.RWMutex
	peerView     map[peer.ID]*peerView

	// disjoint set of blocks which are known but not ready to be processed
	// ie. we only know the hash, number, or the parent block is unknown, or the body is unknown
	// note: the block may have empty fields, as some data about it may be unknown
	pendingBlocks DisjointBlockSet

	state atomic.Value

	finalisedCh <-chan *types.FinalisationInfo

	minPeers     int
	slotDuration time.Duration

	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry
	badBlocks          []string
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
	badBlocks          []string
}

func newChainSync(cfg chainSyncConfig) *chainSync {

	atomicState := atomic.Value{}
	atomicState.Store(bootstrap)
	return &chainSync{
		stopCh:             make(chan struct{}),
		storageState:       cfg.storageState,
		transactionState:   cfg.transactionState,
		babeVerifier:       cfg.babeVerifier,
		finalityGadget:     cfg.finalityGadget,
		blockImportHandler: cfg.blockImportHandler,
		telemetry:          cfg.telemetry,
		blockState:         cfg.bs,
		network:            cfg.net,
		peerView:           make(map[peer.ID]*peerView),
		pendingBlocks:      cfg.pendingBlocks,
		state:              atomicState,
		finalisedCh:        cfg.bs.GetFinalisedNotifierChannel(),
		minPeers:           cfg.minPeers,
		slotDuration:       cfg.slotDuration,
		workerPool:         newSyncWorkerPool(cfg.net),
		badBlocks:          cfg.badBlocks,
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

	isSyncedGauge.Set(0)
	go cs.pendingBlocks.run(cs.finalisedCh, cs.stopCh)
	go cs.workerPool.listenForRequests(cs.stopCh)
	go cs.sync()
}

func (cs *chainSync) stop() {
	close(cs.stopCh)
	<-cs.workerPool.doneCh
}

func (cs *chainSync) sync() {
	for {
		bestBlockHeader, err := cs.blockState.BestBlockHeader()
		if err != nil {
			logger.Criticalf("getting best block header: %s", err)
			return
		}

		syncTarget, err := cs.getTarget()
		if err != nil {
			logger.Criticalf("getting target: %w", err)
			return
		}

		finalisedHeader, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			logger.Criticalf("getting finalised block header: %s", err)
			return
		}
		logger.Infof(
			"🚣 currently syncing, %d peers connected, "+
				"%d available workers, "+
				"target block number %d, "+
				"finalised block number %d with hash %s",
			len(cs.network.Peers()),
			cs.workerPool.totalWorkers(),
			syncTarget, finalisedHeader.Number, finalisedHeader.Hash())

		bestBlockNumber := bestBlockHeader.Number
		isFarFromTarget := bestBlockNumber+maxResponseSize < syncTarget

		if isFarFromTarget {
			// we are at least 128 blocks behind the head, switch to bootstrap
			swapped := cs.state.CompareAndSwap(tip, bootstrap)
			isSyncedGauge.Set(0)

			if swapped {
				logger.Debugf("switched sync mode to %d", bootstrap)
			}

			cs.executeBootstrapSync(finalisedHeader)
		} else {
			// we are less than 128 blocks behind the target we can use tip sync
			swapped := cs.state.CompareAndSwap(bootstrap, tip)
			isSyncedGauge.Set(1)

			if swapped {
				logger.Debugf("switched sync mode to %d", tip)
			}

			cs.requestPendingBlocks(finalisedHeader)
		}
	}
}

func (cs *chainSync) syncState() chainSyncState {
	return cs.state.Load().(chainSyncState)
}

// setPeerHead sets a peer's best known block
func (cs *chainSync) setPeerHead(who peer.ID, bestHash common.Hash, bestNumber uint) {
	cs.workerPool.fromBlockAnnounce(who)

	cs.peerViewLock.Lock()
	defer cs.peerViewLock.Unlock()

	cs.peerView[who] = &peerView{
		who:    who,
		hash:   bestHash,
		number: bestNumber,
	}
}

func (cs *chainSync) onImportBlock(announced announcedBlock) error {
	if cs.pendingBlocks.hasBlock(announced.header.Hash()) {
		return fmt.Errorf("%w: block %s (#%d)",
			errAlreadyInDisjointSet, announced.header.Hash(), announced.header.Number)
	}

	err := cs.pendingBlocks.addHeader(announced.header)
	if err != nil {
		return fmt.Errorf("while adding pending block header: %w", err)
	}

	syncState := cs.state.Load().(chainSyncState)
	switch syncState {
	case tip:
		return cs.requestImportedBlock(announced)
	}

	return nil
}

func (cs *chainSync) requestImportedBlock(announce announcedBlock) error {
	peerWhoAnnounced := announce.who
	announcedHash := announce.header.Hash()
	announcedNumber := announce.header.Number

	has, err := cs.blockState.HasHeader(announcedHash)
	if err != nil {
		return fmt.Errorf("checking if header exists: %s", err)
	}

	if has {
		return nil
	}

	bestBlockHeader, err := cs.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("getting best block header: %w", err)
	}

	// if the announced block contains a lower number than our best
	// block header, let's check if it is greater than our latests
	// finalized header, if so this block belongs to a fork chain
	if announcedNumber < bestBlockHeader.Number {
		highestFinalizedHeader, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return fmt.Errorf("getting highest finalized header")
		}

		// ignore the block if it has the same or lower number
		// TODO: is it following the protocol to send a blockAnnounce with number < highestFinalized number?
		if announcedNumber <= highestFinalizedHeader.Number {
			return nil
		}

		return cs.requestForkBlocks(bestBlockHeader, highestFinalizedHeader, announce.header, announce.who)
	}

	cs.requestChainBlocks(announce.header, bestBlockHeader, peerWhoAnnounced)

	highestFinalizedHeader, err := cs.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("while getting highest finalized header: %w", err)
	}

	err = cs.requestPendingBlocks(highestFinalizedHeader)
	if err != nil {
		return fmt.Errorf("while requesting pending blocks")
	}

	return nil
}

func (cs *chainSync) requestChainBlocks(announcedHeader, bestBlockHeader *types.Header, peerWhoAnnounced peer.ID) {
	gapLength := uint32(announcedHeader.Number - bestBlockHeader.Number)
	startAtBlock := announcedHeader.Number
	totalBlocks := uint32(1)
	var request *network.BlockRequestMessage
	if gapLength > 1 {
		request = descendingBlockRequest(announcedHeader.Hash(), gapLength, bootstrapRequestData)
		startAtBlock = announcedHeader.Number - uint(*request.Max) + 1
		totalBlocks = *request.Max

		logger.Debugf("received a block announce from %s, requesting %d blocks, descending request from %s (#%d)",
			peerWhoAnnounced, gapLength, announcedHeader.Hash(), announcedHeader.Number)
	} else {
		gapLength = 1
		request = singleBlockRequest(announcedHeader.Hash(), bootstrapRequestData)
		logger.Debugf("received a block announce from %s, requesting a single block %s (#%d)",
			peerWhoAnnounced, announcedHeader.Hash(), announcedHeader.Number)
	}

	resultsQueue := make(chan *syncTaskResult)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go cs.handleWorkersResults(resultsQueue, startAtBlock, totalBlocks, &wg)
	cs.workerPool.submitBoundedRequest(request, peerWhoAnnounced, resultsQueue)
	wg.Wait()
}

func (cs *chainSync) requestForkBlocks(bestBlockHeader, highestFinalizedHeader, announcedHeader *types.Header,
	peerWhoAnnounced peer.ID) error {
	logger.Debugf("block announce lower than best block %s (#%d) and greater highest finalized %s (#%d)",
		bestBlockHeader.Hash(), bestBlockHeader.Number, highestFinalizedHeader.Hash(), highestFinalizedHeader.Number)

	parentExists, err := cs.blockState.HasHeader(announcedHeader.ParentHash)
	if err != nil && !errors.Is(err, chaindb.ErrKeyNotFound) {
		return fmt.Errorf("while checking header exists: %w", err)
	}

	gapLength := uint32(1)
	startAtBlock := announcedHeader.Number
	announcedHash := announcedHeader.Hash()
	var request *network.BlockRequestMessage

	if parentExists {
		request = singleBlockRequest(announcedHash, bootstrapRequestData)
	} else {
		gapLength = uint32(announcedHeader.Number - highestFinalizedHeader.Number)
		startAtBlock = highestFinalizedHeader.Number + 1
		request = descendingBlockRequest(announcedHash, gapLength, bootstrapRequestData)
	}

	logger.Debugf("received a block announce from %s, requesting %d blocks, starting %s (#%d)",
		peerWhoAnnounced, gapLength, announcedHash, announcedHeader.Number)

	resultsQueue := make(chan *syncTaskResult)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go cs.handleWorkersResults(resultsQueue, startAtBlock, gapLength, &wg)
	cs.workerPool.submitBoundedRequest(request, peerWhoAnnounced, resultsQueue)
	wg.Wait()

	return nil
}

func (cs *chainSync) requestPendingBlocks(highestFinalizedHeader *types.Header) error {
	logger.Infof("total of pending blocks: %d", cs.pendingBlocks.size())
	if cs.pendingBlocks.size() == 0 {
		return nil
	}

	pendingBlocks := cs.pendingBlocks.getBlocks()
	for _, pendingBlock := range pendingBlocks {
		if pendingBlock.number <= highestFinalizedHeader.Number {
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

		gapLength := pendingBlock.number - highestFinalizedHeader.Number
		if gapLength > 128 {
			logger.Criticalf("GAP LENGHT: %d, GREATER THAN 128 block", gapLength)
			gapLength = 128
		}

		descendingGapRequest := descendingBlockRequest(pendingBlock.hash,
			uint32(gapLength), bootstrapRequestData)
		startAtBlock := pendingBlock.number - uint(*descendingGapRequest.Max) + 1

		// the `requests` in the tip sync are not related necessarily
		// this is why we need to treat them separately
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

func (cs *chainSync) executeBootstrapSync(highestFinalizedHeader *types.Header) error {
	cs.workerPool.useConnectedPeers()

	startRequestAt := highestFinalizedHeader.Number + 1

	const maxRequestsAllowed = 50
	// we build the set of requests based on the amount of available peers
	// in the worker pool, if we have more peers than `maxRequestAllowed`
	// so we limit to `maxRequestAllowed` to avoid the error:
	// cannot reserve outbound connection: resource limit exceeded
	availableWorkers := cs.workerPool.totalWorkers()
	if availableWorkers > maxRequestsAllowed {
		availableWorkers = maxRequestsAllowed
	}

	// targetBlockNumber is the virtual target we will request, however
	// we should bound it to the real target which is collected through
	// block announces received from other peers
	targetBlockNumber := startRequestAt + uint(availableWorkers)*128
	realTarget, err := cs.getTarget()
	if err != nil {
		return fmt.Errorf("while getting target: %w", err)
	}

	if targetBlockNumber > realTarget {
		// basically if our virtual target is beyond the real target
		// that means we are only a few requests away, then we
		// calculate the correct amount of missing requests and then
		// change to tip sync which should take care of the rest
		diff := targetBlockNumber - realTarget
		numOfRequestsToDrop := (diff / 128) + 1
		targetBlockNumber = targetBlockNumber - (numOfRequestsToDrop * 128)
	}

	requests := ascedingBlockRequests(startRequestAt, targetBlockNumber, bootstrapRequestData)
	expectedAmountOfBlocks := totalBlocksRequested(requests)

	wg := sync.WaitGroup{}
	resultsQueue := make(chan *syncTaskResult)

	wg.Add(1)
	go cs.handleWorkersResults(resultsQueue, startRequestAt, expectedAmountOfBlocks, &wg)
	cs.workerPool.submitRequests(requests, resultsQueue)
	wg.Wait()

	return nil
}

// getTarget takes the average of all peer heads
// TODO: should we just return the highest? could be an attack vector potentially, if a peer reports some very large
// head block number, it would leave us in bootstrap mode forever
// it would be better to have some sort of standard deviation calculation and discard any outliers (#1861)
func (cs *chainSync) getTarget() (uint, error) {
	cs.peerViewLock.RLock()
	defer cs.peerViewLock.RUnlock()

	// in practice, this shouldn't happen, as we only start the module once we have some peer states
	if len(cs.peerView) == 0 {
		// return max uint32 instead of 0, as returning 0 would switch us to tip mode unexpectedly
		return 0, errUnableToGetTarget
	}

	// we are going to sort the data and remove the outliers then we will return the avg of all the valid elements
	uintArr := make([]uint, 0, len(cs.peerView))
	for _, ps := range cs.peerView {
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
	startTime := time.Now()
	defer func() {
		totalSyncAndImportSeconds := time.Since(startTime).Seconds()
		bps := float64(totalBlocks) / totalSyncAndImportSeconds
		logger.Debugf("⛓️ synced %d blocks, took: %.2f seconds, bps: %.2f blocks/second", totalBlocks, totalSyncAndImportSeconds, bps)
		wg.Done()
	}()

	logger.Debugf("💤 waiting for %d blocks", totalBlocks)
	syncingChain := make([]*types.BlockData, totalBlocks)
	// the total numbers of blocks is missing in the syncing chain
	waitingBlocks := totalBlocks

	for waitingBlocks > 0 {
		// in a case where we don't handle workers results we should check the pool
		idleDuration := time.Minute
		idleTimer := time.NewTimer(idleDuration)

		select {
		case <-cs.stopCh:
			return

		case <-idleTimer.C:
			logger.Warnf("idle ticker triggered! checking pool")
			cs.workerPool.useConnectedPeers()
			continue

		case taskResult := <-workersResults:
			if !idleTimer.Stop() {
				<-idleTimer.C
			}

			logger.Debugf("task result: peer(%s), with error: %v, with response: %v",
				taskResult.who, taskResult.err != nil, taskResult.response != nil)

			if taskResult.err != nil {
				logger.Errorf("task result: peer(%s) error: %s",
					taskResult.who, taskResult.err)

				if !errors.Is(taskResult.err, network.ErrReceivedEmptyMessage) {
					switch {
					case strings.Contains(taskResult.err.Error(), "protocols not supported"):
						cs.network.ReportPeer(peerset.ReputationChange{
							Value:  peerset.BadProtocolValue,
							Reason: peerset.BadProtocolReason,
						}, taskResult.who)
						cs.workerPool.ignorePeerAsWorker(taskResult.who)
					default:
						cs.workerPool.punishPeer(taskResult.who)
					}
				}

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
				cs.workerPool.punishPeer(taskResult.who)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			case errors.Is(err, errEmptyBlockData):
				cs.workerPool.punishPeer(taskResult.who)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			case errors.Is(err, errUnknownParent):
			case errors.Is(err, errBadBlock):
				logger.Warnf("peer %s sent a bad block: %s", who, err)
				cs.workerPool.ignorePeerAsWorker(taskResult.who)
				cs.workerPool.submitRequest(taskResult.request, workersResults)
				continue
			case err != nil:
				logger.Criticalf("response invalid: %s", err)
				cs.workerPool.punishPeer(taskResult.who)
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

			cs.network.ReportPeer(peerset.ReputationChange{
				Value:  peerset.GossipSuccessValue,
				Reason: peerset.GossipSuccessReason,
			}, taskResult.who)

			for _, blockInResponse := range response.BlockData {
				blockExactIndex := blockInResponse.Header.Number - startAtBlock
				syncingChain[blockExactIndex] = blockInResponse
			}

			// we need to check if we've filled all positions
			// otherwise we should wait for more responses
			waitingBlocks -= uint32(len(response.BlockData))
		}
	}

	retreiveBlocksSeconds := time.Since(startTime).Seconds()
	logger.Debugf("🔽 retrieved %d blocks, took: %.2f seconds, starting process...", totalBlocks, retreiveBlocksSeconds)
	if len(syncingChain) >= 2 {
		// ensuring the parents are in the right place
		parentElement := syncingChain[0]
		for _, element := range syncingChain[1:] {
			if parentElement.Header.Hash() != element.Header.ParentHash {
				panic(fmt.Sprintf("expected %s be parent of %s",
					parentElement.Header.Hash(), element.Header.ParentHash))
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

		if block.header == nil {
			logger.Errorf("new ready block number (unknown) with hash %s", bd.Hash)
			return nil
		}

		bd.Header = block.header
	}

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
	headerInState, err := cs.blockState.HasHeader(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has header: %w", err)
	}

	bodyInState, err := cs.blockState.HasBlockBody(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has body: %w", err)
	}

	// while in bootstrap mode we don't need to broadcast block announcements
	announceImportedBlock := cs.state.Load().(chainSyncState) == tip
	if headerInState && bodyInState {
		err = cs.processBlockDataWithStateHeaderAndBody(blockData, announceImportedBlock)
		if err != nil {
			return fmt.Errorf("processing block data with header and "+
				"body in block state: %w", err)
		}
		return nil
	}

	if blockData.Header != nil {
		if blockData.Body != nil {
			err = cs.processBlockDataWithHeaderAndBody(blockData, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			logger.Infof("handling justification for block %s (#%d)", blockData.Hash.Short(), blockData.Number())
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
//   - the block is not contained in the bad block list
//   - each block has the correct parent, ie. the response constitutes a valid chain
func (cs *chainSync) validateResponse(req *network.BlockRequestMessage,
	resp *network.BlockResponseMessage, p peer.ID) error {
	if resp == nil || len(resp.BlockData) == 0 {
		return errEmptyBlockData
	}

	logger.Tracef("validating block response starting at block hash %s", resp.BlockData[0].Hash)

	headerRequested := (req.RequestedData & network.RequestedDataHeader) == 1
	firstItem := resp.BlockData[0]

	has, err := cs.blockState.HasHeader(firstItem.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("while checking ancestry: %w", err)
	}

	if !has {
		return errUnknownParent
	}

	previousBlockData := firstItem
	for _, currBlockData := range resp.BlockData[1:] {
		if err := cs.validateBlockData(req, currBlockData, p); err != nil {
			return err
		}

		if headerRequested {
			previousHash := previousBlockData.Header.Hash()
			if previousHash != currBlockData.Header.ParentHash ||
				currBlockData.Header.Number != (previousBlockData.Header.Number+1) {
				return errResponseIsNotChain
			}
		} else if currBlockData.Justification != nil {
			// if this is a justification-only request, make sure we have the block for the justification
			has, _ := cs.blockState.HasHeader(currBlockData.Hash)
			if !has {
				cs.network.ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadJustificationValue,
					Reason: peerset.BadJustificationReason,
				}, p)
				return errUnknownBlockForJustification
			}
		}

		previousBlockData = currBlockData
	}

	return nil
}

// validateBlockData checks that the expected fields are in the block data
func (cs *chainSync) validateBlockData(req *network.BlockRequestMessage, bd *types.BlockData, p peer.ID) error {
	if bd == nil {
		return errNilBlockData
	}

	requestedData := req.RequestedData

	if slices.Contains(cs.badBlocks, bd.Hash.String()) {
		logger.Errorf("Rejecting known bad block Number: %d Hash: %s", bd.Number(), bd.Hash)
		return errBadBlock
	}

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

func (cs *chainSync) getHighestBlock() (highestBlock uint, err error) {
	cs.peerViewLock.RLock()
	defer cs.peerViewLock.RUnlock()

	if len(cs.peerView) == 0 {
		return 0, errNoPeers
	}

	for _, ps := range cs.peerView {
		if ps.number < highestBlock {
			continue
		}
		highestBlock = ps.number
	}

	return highestBlock, nil
}
