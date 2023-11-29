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

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"
)

var _ ChainSync = (*chainSync)(nil)

type chainSyncState byte

const (
	bootstrap chainSyncState = iota
	tip
)

type blockOrigin byte

const (
	networkInitialSync blockOrigin = iota
	networkBroadcast
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
	pendingBlocksLimit = network.MaxBlocksInResponse * 32
	isSyncedGauge      = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_network_syncer",
		Name:      "is_synced",
		Help:      "bool representing whether the node is synced to the head of the chain",
	})

	blockSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_sync",
		Name:      "block_size",
		Help:      "represent the size of blocks synced",
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
	stop() error

	// called upon receiving a BlockAnnounceHandshake
	onBlockAnnounceHandshake(p peer.ID, hash common.Hash, number uint) error

	// getSyncMode returns the current syncing state
	getSyncMode() chainSyncState

	// getHighestBlock returns the highest block or an error
	getHighestBlock() (highestBlock uint, err error)

	onBlockAnnounce(announcedBlock) error
}

type announcedBlock struct {
	who    peer.ID
	header *types.Header
}

type chainSync struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	blockState BlockState
	network    Network

	workerPool *syncWorkerPool

	// tracks the latest state we know of from our peers,
	// ie. their best block hash and number
	peerViewLock sync.RWMutex
	peerView     map[peer.ID]peerView

	// disjoint set of blocks which are known but not ready to be processed
	// ie. we only know the hash, number, or the parent block is unknown, or the body is unknown
	// note: the block may have empty fields, as some data about it may be unknown
	pendingBlocks DisjointBlockSet

	syncMode atomic.Value

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
	requestMaker       network.RequestMaker
	waitPeersDuration  time.Duration
}

type chainSyncConfig struct {
	bs                 BlockState
	net                Network
	requestMaker       network.RequestMaker
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
	waitPeersDuration  time.Duration
}

func newChainSync(cfg chainSyncConfig) *chainSync {
	atomicState := atomic.Value{}
	atomicState.Store(tip)
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
		peerView:           make(map[peer.ID]peerView),
		pendingBlocks:      cfg.pendingBlocks,
		syncMode:           atomicState,
		finalisedCh:        cfg.bs.GetFinalisedNotifierChannel(),
		minPeers:           cfg.minPeers,
		slotDuration:       cfg.slotDuration,
		workerPool:         newSyncWorkerPool(cfg.net, cfg.requestMaker),
		badBlocks:          cfg.badBlocks,
		requestMaker:       cfg.requestMaker,
		waitPeersDuration:  cfg.waitPeersDuration,
	}
}

func (cs *chainSync) waitEnoughPeersAndTarget() {
	waitPeersTimer := time.NewTimer(cs.waitPeersDuration)

	for {
		cs.workerPool.useConnectedPeers()
		_, err := cs.getTarget()

		totalAvailable := cs.workerPool.totalWorkers()
		if totalAvailable >= uint(cs.minPeers) && err == nil {
			return
		}

		select {
		case <-waitPeersTimer.C:
			waitPeersTimer.Reset(cs.waitPeersDuration)
		case <-cs.stopCh:
			return
		}
	}
}

func (cs *chainSync) start() {
	// since the default status from sync mode is syncMode(tip)
	isSyncedGauge.Set(1)

	cs.wg.Add(1)
	go cs.pendingBlocks.run(cs.finalisedCh, cs.stopCh, &cs.wg)

	// wait until we have a minimal workers in the sync worker pool
	cs.waitEnoughPeersAndTarget()
}

func (cs *chainSync) stop() error {
	err := cs.workerPool.stop()
	if err != nil {
		return fmt.Errorf("stopping worker poll: %w", err)
	}

	close(cs.stopCh)
	allStopCh := make(chan struct{})
	go func() {
		defer close(allStopCh)
		cs.wg.Wait()
	}()

	timeoutTimer := time.NewTimer(30 * time.Second)

	select {
	case <-allStopCh:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}
		return nil
	case <-timeoutTimer.C:
		return ErrStopTimeout
	}
}

func (cs *chainSync) isBootstrap() (bestBlockHeader *types.Header, syncTarget uint,
	isBootstrap bool, err error) {
	syncTarget, err = cs.getTarget()
	if err != nil {
		return nil, syncTarget, false, fmt.Errorf("getting target: %w", err)
	}

	bestBlockHeader, err = cs.blockState.BestBlockHeader()
	if err != nil {
		return nil, syncTarget, false, fmt.Errorf("getting best block header: %w", err)
	}

	bestBlockNumber := bestBlockHeader.Number
	isBootstrap = bestBlockNumber+network.MaxBlocksInResponse < syncTarget
	return bestBlockHeader, syncTarget, isBootstrap, nil
}

func (cs *chainSync) bootstrapSync() {
	defer cs.wg.Done()

	for {
		select {
		case <-cs.stopCh:
			logger.Warn("ending bootstrap sync, chain sync stop channel triggered")
			return
		default:
		}

		bestBlockHeader, syncTarget, isFarFromTarget, err := cs.isBootstrap()
		if err != nil && !errors.Is(err, errNoPeerViews) {
			logger.Criticalf("ending bootstrap sync, checking target distance: %s", err)
			return
		}

		finalisedHeader, err := cs.blockState.GetHighestFinalisedHeader()
		if err != nil {
			logger.Criticalf("getting highest finalized header: %w", err)
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

		if isFarFromTarget {
			cs.workerPool.useConnectedPeers()
			err = cs.requestMaxBlocksFrom(bestBlockHeader, networkInitialSync)
			if err != nil {
				logger.Errorf("requesting max blocks from best block header: %s", err)
			}
		} else {
			// we are less than 128 blocks behind the target we can use tip sync
			cs.syncMode.Store(tip)
			isSyncedGauge.Set(1)
			logger.Debugf("switched sync mode to %s", tip.String())
			return
		}
	}
}

func (cs *chainSync) getSyncMode() chainSyncState {
	return cs.syncMode.Load().(chainSyncState)
}

// onBlockAnnounceHandshake sets a peer's best known block
func (cs *chainSync) onBlockAnnounceHandshake(who peer.ID, bestHash common.Hash, bestNumber uint) error {
	cs.workerPool.fromBlockAnnounce(who)

	cs.peerViewLock.Lock()
	cs.peerView[who] = peerView{
		who:    who,
		hash:   bestHash,
		number: bestNumber,
	}
	cs.peerViewLock.Unlock()

	if cs.getSyncMode() == bootstrap {
		return nil
	}

	_, _, isFarFromTarget, err := cs.isBootstrap()
	if err != nil && !errors.Is(err, errNoPeerViews) {
		return fmt.Errorf("checking target distance: %w", err)
	}

	if !isFarFromTarget {
		return nil
	}

	// we are more than 128 blocks behind the head, switch to bootstrap
	cs.syncMode.Store(bootstrap)
	isSyncedGauge.Set(0)
	logger.Debugf("switched sync mode to %s", bootstrap.String())

	cs.wg.Add(1)
	go cs.bootstrapSync()
	return nil
}

func (cs *chainSync) onBlockAnnounce(announced announcedBlock) error {
	// TODO: https://github.com/ChainSafe/gossamer/issues/3432
	cs.workerPool.fromBlockAnnounce(announced.who)

	if cs.pendingBlocks.hasBlock(announced.header.Hash()) {
		return fmt.Errorf("%w: block %s (#%d)",
			errAlreadyInDisjointSet, announced.header.Hash(), announced.header.Number)
	}

	err := cs.pendingBlocks.addHeader(announced.header)
	if err != nil {
		return fmt.Errorf("while adding pending block header: %w", err)
	}

	if cs.getSyncMode() == bootstrap {
		return nil
	}

	_, _, isFarFromTarget, err := cs.isBootstrap()
	if err != nil && !errors.Is(err, errNoPeerViews) {
		return fmt.Errorf("checking target distance: %w", err)
	}

	if !isFarFromTarget {
		return cs.requestAnnouncedBlock(announced)
	}

	return nil
}

func (cs *chainSync) requestAnnouncedBlock(announce announcedBlock) error {
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

	highestFinalizedHeader, err := cs.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("getting highest finalized header")
	}

	// if the announced block contains a lower number than our best
	// block header, let's check if it is greater than our latests
	// finalized header, if so this block belongs to a fork chain
	if announcedNumber < bestBlockHeader.Number {
		// ignore the block if it has the same or lower number
		// TODO: is it following the protocol to send a blockAnnounce with number < highestFinalized number?
		if announcedNumber <= highestFinalizedHeader.Number {
			return nil
		}

		return cs.requestForkBlocks(bestBlockHeader, highestFinalizedHeader, announce.header, announce.who)
	}

	err = cs.requestChainBlocks(announce.header, bestBlockHeader, peerWhoAnnounced)
	if err != nil {
		return fmt.Errorf("requesting chain blocks: %w", err)
	}

	err = cs.requestPendingBlocks(highestFinalizedHeader)
	if err != nil {
		return fmt.Errorf("while requesting pending blocks")
	}

	return nil
}

func (cs *chainSync) requestChainBlocks(announcedHeader, bestBlockHeader *types.Header,
	peerWhoAnnounced peer.ID) error {
	gapLength := uint32(announcedHeader.Number - bestBlockHeader.Number)
	startAtBlock := announcedHeader.Number
	totalBlocks := uint32(1)

	var request *network.BlockRequestMessage
	startingBlock := *variadic.MustNewUint32OrHash(announcedHeader.Hash())

	if gapLength > 1 {
		request = network.NewBlockRequest(startingBlock, gapLength,
			network.BootstrapRequestData, network.Descending)

		startAtBlock = announcedHeader.Number - uint(*request.Max) + 1
		totalBlocks = *request.Max

		logger.Debugf("received a block announce from %s, requesting %d blocks, descending request from %s (#%d)",
			peerWhoAnnounced, gapLength, announcedHeader.Hash(), announcedHeader.Number)
	} else {
		request = network.NewBlockRequest(startingBlock, 1, network.BootstrapRequestData, network.Descending)
		logger.Debugf("received a block announce from %s, requesting a single block %s (#%d)",
			peerWhoAnnounced, announcedHeader.Hash(), announcedHeader.Number)
	}

	resultsQueue := make(chan *syncTaskResult)
	cs.workerPool.submitRequest(request, &peerWhoAnnounced, resultsQueue)
	err := cs.handleWorkersResults(resultsQueue, networkBroadcast, startAtBlock, totalBlocks)
	if err != nil {
		return fmt.Errorf("while handling workers results: %w", err)
	}

	return nil
}

func (cs *chainSync) requestForkBlocks(bestBlockHeader, highestFinalizedHeader, announcedHeader *types.Header,
	peerWhoAnnounced peer.ID) error {
	logger.Debugf("block announce lower than best block %s (#%d) and greater highest finalized %s (#%d)",
		bestBlockHeader.Hash(), bestBlockHeader.Number, highestFinalizedHeader.Hash(), highestFinalizedHeader.Number)

	parentExists, err := cs.blockState.HasHeader(announcedHeader.ParentHash)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("while checking header exists: %w", err)
	}

	gapLength := uint32(1)
	startAtBlock := announcedHeader.Number
	announcedHash := announcedHeader.Hash()
	var request *network.BlockRequestMessage
	startingBlock := *variadic.MustNewUint32OrHash(announcedHash)

	if parentExists {
		request = network.NewBlockRequest(startingBlock, 1, network.BootstrapRequestData, network.Descending)
	} else {
		gapLength = uint32(announcedHeader.Number - highestFinalizedHeader.Number)
		startAtBlock = highestFinalizedHeader.Number + 1
		request = network.NewBlockRequest(startingBlock, gapLength, network.BootstrapRequestData, network.Descending)
	}

	logger.Debugf("requesting %d fork blocks, starting at %s (#%d)",
		peerWhoAnnounced, gapLength, announcedHash, announcedHeader.Number)

	resultsQueue := make(chan *syncTaskResult)
	cs.workerPool.submitRequest(request, &peerWhoAnnounced, resultsQueue)

	err = cs.handleWorkersResults(resultsQueue, networkBroadcast, startAtBlock, gapLength)
	if err != nil {
		return fmt.Errorf("while handling workers results: %w", err)
	}

	return nil
}

func (cs *chainSync) requestPendingBlocks(highestFinalizedHeader *types.Header) error {
	pendingBlocksTotal := cs.pendingBlocks.size()
	logger.Infof("total of pending blocks: %d", pendingBlocksTotal)
	if pendingBlocksTotal < 1 {
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
			err := cs.handleReadyBlock(pendingBlock.toBlockData(), networkBroadcast)
			if err != nil {
				return fmt.Errorf("handling ready block: %w", err)
			}
			continue
		}

		gapLength := pendingBlock.number - highestFinalizedHeader.Number
		if gapLength > 128 {
			logger.Warnf("gap of %d blocks, max expected: 128 block", gapLength)
			gapLength = 128
		}

		descendingGapRequest := network.NewBlockRequest(*variadic.MustNewUint32OrHash(pendingBlock.hash),
			uint32(gapLength), network.BootstrapRequestData, network.Descending)
		startAtBlock := pendingBlock.number - uint(*descendingGapRequest.Max) + 1

		// the `requests` in the tip sync are not related necessarily
		// this is why we need to treat them separately
		resultsQueue := make(chan *syncTaskResult)
		cs.workerPool.submitRequest(descendingGapRequest, nil, resultsQueue)

		// TODO: we should handle the requests concurrently
		// a way of achieve that is by constructing a new `handleWorkersResults` for
		// handling only tip sync requests
		err = cs.handleWorkersResults(resultsQueue, networkBroadcast, startAtBlock, *descendingGapRequest.Max)
		if err != nil {
			return fmt.Errorf("while handling workers results: %w", err)
		}
	}

	return nil
}

func (cs *chainSync) requestMaxBlocksFrom(bestBlockHeader *types.Header, origin blockOrigin) error { //nolint:unparam
	startRequestAt := bestBlockHeader.Number + 1

	// targetBlockNumber is the virtual target we will request, however
	// we should bound it to the real target which is collected through
	// block announces received from other peers
	targetBlockNumber := startRequestAt + maxRequestsAllowed*128
	realTarget, err := cs.getTarget()
	if err != nil {
		return fmt.Errorf("while getting target: %w", err)
	}

	if targetBlockNumber > realTarget {
		targetBlockNumber = realTarget
	}

	requests := network.NewAscendingBlockRequests(startRequestAt, targetBlockNumber,
		network.BootstrapRequestData)

	var expectedAmountOfBlocks uint32
	for _, request := range requests {
		if request.Max != nil {
			expectedAmountOfBlocks += *request.Max
		}
	}

	resultsQueue := cs.workerPool.submitRequests(requests)
	err = cs.handleWorkersResults(resultsQueue, origin, startRequestAt, expectedAmountOfBlocks)
	if err != nil {
		return fmt.Errorf("while handling workers results: %w", err)
	}

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
		return 0, errNoPeerViews
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
// TODO: handle only justification requests
func (cs *chainSync) handleWorkersResults(
	workersResults chan *syncTaskResult, origin blockOrigin, startAtBlock uint, expectedSyncedBlocks uint32) error {

	startTime := time.Now()
	defer func() {
		totalSyncAndImportSeconds := time.Since(startTime).Seconds()
		bps := float64(expectedSyncedBlocks) / totalSyncAndImportSeconds
		logger.Debugf("⛓️ synced %d blocks, "+
			"took: %.2f seconds, bps: %.2f blocks/second",
			expectedSyncedBlocks, totalSyncAndImportSeconds, bps)
	}()

	syncingChain := make([]*types.BlockData, expectedSyncedBlocks)
	// the total numbers of blocks is missing in the syncing chain
	waitingBlocks := expectedSyncedBlocks

taskResultLoop:
	for waitingBlocks > 0 {
		// in a case where we don't handle workers results we should check the pool
		idleDuration := time.Minute
		idleTimer := time.NewTimer(idleDuration)

		select {
		case <-cs.stopCh:
			return nil

		case <-idleTimer.C:
			logger.Warnf("idle ticker triggered! checking pool")
			cs.workerPool.useConnectedPeers()
			continue

		case taskResult := <-workersResults:
			if !idleTimer.Stop() {
				<-idleTimer.C
			}

			who := taskResult.who
			request := taskResult.request
			response := taskResult.response

			logger.Debugf("task result: peer(%s), with error: %v, with response: %v",
				taskResult.who, taskResult.err != nil, taskResult.response != nil)

			if taskResult.err != nil {
				logger.Errorf("task result: peer(%s) error: %s",
					taskResult.who, taskResult.err)

				if !errors.Is(taskResult.err, network.ErrReceivedEmptyMessage) {
					if strings.Contains(taskResult.err.Error(), "protocols not supported") {
						cs.network.ReportPeer(peerset.ReputationChange{
							Value:  peerset.BadProtocolValue,
							Reason: peerset.BadProtocolReason,
						}, who)
					}
				}

				// TODO: avoid the same peer to get the same task
				cs.workerPool.submitRequest(request, nil, workersResults)
				continue
			}

			if request.Direction == network.Descending {
				// reverse blocks before pre-validating and placing in ready queue
				reverseBlockData(response.BlockData)
			}

			err := validateResponseFields(request.RequestedData, response.BlockData)
			if err != nil {
				logger.Criticalf("validating fields: %s", err)
				// TODO: check the reputation change for nil body in response
				// and nil justification in response
				if errors.Is(err, errNilHeaderInResponse) {
					cs.network.ReportPeer(peerset.ReputationChange{
						Value:  peerset.IncompleteHeaderValue,
						Reason: peerset.IncompleteHeaderReason,
					}, who)
				}

				cs.workerPool.submitRequest(taskResult.request, nil, workersResults)
				continue taskResultLoop
			}

			isChain := isResponseAChain(response.BlockData)
			if !isChain {
				logger.Criticalf("response from %s is not a chain", who)
				cs.workerPool.submitRequest(taskResult.request, nil, workersResults)
				continue taskResultLoop
			}

			grows := doResponseGrowsTheChain(response.BlockData, syncingChain,
				startAtBlock, expectedSyncedBlocks)
			if !grows {
				logger.Criticalf("response from %s does not grows the ongoing chain", who)
				cs.workerPool.submitRequest(taskResult.request, nil, workersResults)
				continue taskResultLoop
			}

			for _, blockInResponse := range response.BlockData {
				if slices.Contains(cs.badBlocks, blockInResponse.Hash.String()) {
					logger.Criticalf("%s sent a known bad block: %s (#%d)",
						who, blockInResponse.Hash.String(), blockInResponse.Number())

					cs.network.ReportPeer(peerset.ReputationChange{
						Value:  peerset.BadBlockAnnouncementValue,
						Reason: peerset.BadBlockAnnouncementReason,
					}, who)

					cs.workerPool.ignorePeerAsWorker(taskResult.who)
					cs.workerPool.submitRequest(taskResult.request, nil, workersResults)
					continue taskResultLoop
				}

				blockExactIndex := blockInResponse.Header.Number - startAtBlock
				syncingChain[blockExactIndex] = blockInResponse
			}

			// we need to check if we've filled all positions
			// otherwise we should wait for more responses
			waitingBlocks -= uint32(len(response.BlockData))

			// we received a response without the desired amount of blocks
			// we should include a new request to retrieve the missing blocks
			if len(response.BlockData) < int(*request.Max) {
				difference := uint32(int(*request.Max) - len(response.BlockData))
				lastItem := response.BlockData[len(response.BlockData)-1]

				startRequestNumber := uint32(lastItem.Header.Number + 1)
				startAt, err := variadic.NewUint32OrHash(startRequestNumber)
				if err != nil {
					panic(err)
				}

				taskResult.request = &network.BlockRequestMessage{
					RequestedData: network.BootstrapRequestData,
					StartingBlock: *startAt,
					Direction:     network.Ascending,
					Max:           &difference,
				}
				cs.workerPool.submitRequest(taskResult.request, nil, workersResults)
				continue taskResultLoop
			}
		}
	}

	retreiveBlocksSeconds := time.Since(startTime).Seconds()
	logger.Debugf("🔽 retrieved %d blocks, took: %.2f seconds, starting process...",
		expectedSyncedBlocks, retreiveBlocksSeconds)

	// response was validated! place into ready block queue
	for _, bd := range syncingChain {
		// block is ready to be processed!
		if err := cs.handleReadyBlock(bd, origin); err != nil {
			return fmt.Errorf("while handling ready block: %w", err)
		}
	}

	return nil
}

func (cs *chainSync) handleReadyBlock(bd *types.BlockData, origin blockOrigin) error {
	// if header was not requested, get it from the pending set
	// if we're expecting headers, validate should ensure we have a header
	if bd.Header == nil {
		block := cs.pendingBlocks.getBlock(bd.Hash)
		if block == nil {
			// block wasn't in the pending set!
			// let's check the db as maybe we already processed it
			has, err := cs.blockState.HasHeader(bd.Hash)
			if err != nil && !errors.Is(err, database.ErrNotFound) {
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

	err := cs.processBlockData(*bd, origin)
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
func (cs *chainSync) processBlockData(blockData types.BlockData, origin blockOrigin) error {
	// while in bootstrap mode we don't need to broadcast block announcements
	announceImportedBlock := cs.getSyncMode() == tip
	var blockDataJustification []byte
	if blockData.Justification != nil {
		blockDataJustification = *blockData.Justification
	}

	if blockData.Header != nil {
		round, setID, err := cs.verifyJustification(blockData.Header.Hash(), blockDataJustification)
		if err != nil {
			return err
		}

		if blockData.Body != nil {
			err = cs.processBlockDataWithHeaderAndBody(blockData, origin, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		err = cs.finalizeAndSetJustification(
			blockData.Header,
			round, setID,
			blockDataJustification)
		if err != nil {
			return fmt.Errorf("while setting justification: %w", err)
		}
	}

	err := cs.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (cs *chainSync) verifyJustification(headerHash common.Hash, justification []byte) (
	round uint64, setID uint64, err error) {
	if len(justification) > 0 {
		round, setID, err = cs.finalityGadget.VerifyBlockJustification(headerHash, justification)
		return round, setID, err
	}

	return 0, 0, nil
}

func (cs *chainSync) processBlockDataWithHeaderAndBody(blockData types.BlockData,
	origin blockOrigin, announceImportedBlock bool) (err error) {

	if origin != networkInitialSync {
		err = cs.babeVerifier.VerifyBlock(blockData.Header)
		if err != nil {
			return fmt.Errorf("babe verifying block: %w", err)
		}
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
	acc := 0
	for _, ext := range *body {
		acc += len(ext)
		cs.transactionState.RemoveExtrinsic(ext)
	}

	blockSizeGauge.Set(float64(acc))
}

func (cs *chainSync) finalizeAndSetJustification(header *types.Header,
	round, setID uint64, justification []byte) (err error) {
	if len(justification) > 0 {
		err = cs.blockState.SetFinalisedHash(header.Hash(), round, setID)
		if err != nil {
			return fmt.Errorf("setting finalised hash: %w", err)
		}

		logger.Debugf(
			"finalised block with hash #%d (%s), round %d and set id %d",
			header.Number, header.Hash(), round, setID)

		err = cs.blockState.SetJustification(header.Hash(), justification)
		if err != nil {
			return fmt.Errorf("setting justification for block number %d: %w",
				header.Number, err)
		}

		logger.Infof("🔨 finalised block number #%d (%s)", header.Number, header.Hash())
	}

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

	root := ts.MustRoot(trie.NoMaxInlineValueSize)
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

// validateResponseFields checks that the expected fields are in the block data
func validateResponseFields(requestedData byte, blocks []*types.BlockData) error {
	for _, bd := range blocks {
		if bd == nil {
			return errNilBlockData
		}

		if (requestedData&network.RequestedDataHeader) == network.RequestedDataHeader && bd.Header == nil {
			return fmt.Errorf("%w: %s", errNilHeaderInResponse, bd.Hash)
		}

		if (requestedData&network.RequestedDataBody) == network.RequestedDataBody && bd.Body == nil {
			return fmt.Errorf("%w: %s", errNilBodyInResponse, bd.Hash)
		}

		// if we requested strictly justification
		if (requestedData|network.RequestedDataJustification) == network.RequestedDataJustification &&
			bd.Justification == nil {
			return fmt.Errorf("%w: %s", errNilJustificationInResponse, bd.Hash)
		}
	}

	return nil
}

func isResponseAChain(responseBlockData []*types.BlockData) bool {
	if len(responseBlockData) < 2 {
		return true
	}

	previousBlockData := responseBlockData[0]
	for _, currBlockData := range responseBlockData[1:] {
		previousHash := previousBlockData.Header.Hash()
		isParent := previousHash == currBlockData.Header.ParentHash
		if !isParent {
			return false
		}

		previousBlockData = currBlockData
	}

	return true
}

// doResponseGrowsTheChain will check if the acquired blocks grows the current chain
// matching their parent hashes
func doResponseGrowsTheChain(response, ongoingChain []*types.BlockData, startAtBlock uint, expectedTotal uint32) bool {
	// the ongoing chain does not have any element, we can safely insert an item in it
	if len(ongoingChain) < 1 {
		return true
	}

	compareParentHash := func(parent, child *types.BlockData) bool {
		return parent.Header.Hash() == child.Header.ParentHash
	}

	firstBlockInResponse := response[0]
	firstBlockExactIndex := firstBlockInResponse.Header.Number - startAtBlock
	if firstBlockExactIndex != 0 {
		leftElement := ongoingChain[firstBlockExactIndex-1]
		if leftElement != nil && !compareParentHash(leftElement, firstBlockInResponse) {
			return false
		}

	}

	switch {
	// if the response contains only one block then we should check both sides
	// for example, if the response contains only one block called X we should
	// check if its parent hash matches with the left element as well as we should
	// check if the right element contains X hash as its parent hash
	// ... W <- X -> Y ...
	// we can skip left side comparison if X is in the 0 index and we can skip
	// right side comparison if X is in the last index
	case len(response) == 1:
		if uint32(firstBlockExactIndex+1) < expectedTotal {
			rightElement := ongoingChain[firstBlockExactIndex+1]
			if rightElement != nil && !compareParentHash(firstBlockInResponse, rightElement) {
				return false
			}
		}
	// if the response contains more than 1 block then we need to compare
	// only the start and the end of the acquired response, for example
	// let's say we receive a response [C, D, E] and we need to check
	// if those values fits correctly:
	// ... B <- C D E -> F
	// we skip the left check if its index is equals to 0 and we skip the right
	// check if it ends in the latest position of the ongoing array
	case len(response) > 1:
		lastBlockInResponse := response[len(response)-1]
		lastBlockExactIndex := lastBlockInResponse.Header.Number - startAtBlock

		if uint32(lastBlockExactIndex+1) < expectedTotal {
			rightElement := ongoingChain[lastBlockExactIndex+1]
			if rightElement != nil && !compareParentHash(lastBlockInResponse, rightElement) {
				return false
			}
		}
	}

	return true
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
