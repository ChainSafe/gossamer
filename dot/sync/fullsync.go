// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"container/list"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const defaultNumOfTasks = 3

var _ Strategy = (*FullSyncStrategy)(nil)

var (
	errFailedToGetParent   = errors.New("failed to get parent header")
	errNilHeaderInResponse = errors.New("expected header, received none")
	errNilBodyInResponse   = errors.New("expected body, received none")
	errPeerOnInvalidFork   = errors.New("peer is on an invalid fork")

	blockSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_sync",
		Name:      "block_size",
		Help:      "represent the size of blocks synced",
	})
)

// Config is the configuration for the sync Service.
type FullSyncConfig struct {
	StorageState       StorageState
	TransactionState   TransactionState
	BabeVerifier       BabeVerifier
	FinalityGadget     FinalityGadget
	BlockImportHandler BlockImportHandler
	Telemetry          Telemetry
	BlockState         BlockState
	BadBlocks          []string
	NumOfTasks         int
	RequestMaker       network.RequestMaker
}

type Importer interface {
	handle(*types.BlockData, BlockOrigin) (imported bool, err error)
}

// FullSyncStrategy protocol is the "default" protocol.
// Full sync works by listening to announced blocks and requesting the blocks
// from the announcing peers.
type FullSyncStrategy struct {
	requestQueue  *requestsQueue[*messages.BlockRequestMessage]
	unreadyBlocks *unreadyBlocks
	peers         *peerViewSet
	badBlocks     []string
	reqMaker      network.RequestMaker
	blockState    BlockState
	numOfTasks    int
	startedAt     time.Time
	syncedBlocks  int
	importer      Importer
}

func NewFullSyncStrategy(cfg *FullSyncConfig) *FullSyncStrategy {
	if cfg.NumOfTasks == 0 {
		cfg.NumOfTasks = defaultNumOfTasks
	}

	return &FullSyncStrategy{
		badBlocks:     cfg.BadBlocks,
		reqMaker:      cfg.RequestMaker,
		blockState:    cfg.BlockState,
		numOfTasks:    cfg.NumOfTasks,
		importer:      newBlockImporter(cfg),
		unreadyBlocks: newUnreadyBlocks(),
		requestQueue: &requestsQueue[*messages.BlockRequestMessage]{
			queue: list.New(),
		},
		peers: &peerViewSet{
			view:   make(map[peer.ID]peerView),
			target: 0,
		},
	}
}

func (f *FullSyncStrategy) NextActions() ([]*syncTask, error) {
	f.startedAt = time.Now()
	f.syncedBlocks = 0

	messagesToSend := []*messages.BlockRequestMessage{}
	for f.requestQueue.Len() > 0 {
		msg, ok := f.requestQueue.PopFront()
		if ok {
			messagesToSend = append(messagesToSend, msg)
		}
	}

	currentTarget := f.peers.getTarget()
	bestBlockHeader, err := f.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("while getting best block header")
	}

	// our best block is equal or ahead of current target.
	// in the node's pov we are not legging behind so there's nothing to do
	// or we didn't receive block announces, so lets ask for more blocks
	if uint32(bestBlockHeader.Number) >= currentTarget {
		ascendingBlockRequests := messages.NewBlockRequest(
			*variadic.Uint32OrHashFrom(bestBlockHeader.Hash()),
			messages.MaxBlocksInResponse,
			messages.BootstrapRequestData,
			messages.Ascending,
		)

		messagesToSend = append(messagesToSend, ascendingBlockRequests)
		return f.createTasks(messagesToSend), nil
	}

	startRequestAt := bestBlockHeader.Number + 1
	targetBlockNumber := startRequestAt + maxRequestsAllowed*127

	if targetBlockNumber > uint(currentTarget) {
		targetBlockNumber = uint(currentTarget)
	}

	ascendingBlockRequests := messages.NewAscendingBlockRequests(
		uint32(startRequestAt), uint32(targetBlockNumber),
		messages.BootstrapRequestData)

	return f.createTasks(ascendingBlockRequests), nil
}

func (f *FullSyncStrategy) createTasks(requests []*messages.BlockRequestMessage) []*syncTask {
	tasks := make([]*syncTask, len(requests))
	for idx, req := range requests {
		tasks[idx] = &syncTask{
			request:      req,
			response:     &messages.BlockResponseMessage{},
			requestMaker: f.reqMaker,
		}
	}
	return tasks
}

func (f *FullSyncStrategy) IsFinished(results []*syncTaskResult) (bool, []Change, []peer.ID, error) {
	repChanges, peersToIgnore, validResp := validateResults(results, f.badBlocks)
	logger.Debugf("evaluating %d task results, %d valid responses", len(results), len(validResp))

	var highestFinalized *types.Header
	highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return false, nil, nil, fmt.Errorf("getting highest finalized header")
	}

	readyBlocks := make([][]*types.BlockData, 0, len(validResp))
	for _, reqRespData := range validResp {
		// if Gossamer requested the header, then the response data should
		// contains the full blocks to be imported.
		// if Gossamer didn't request the header, then the response should
		// only contain the missing parts that will complete the unreadyBlocks
		// and then with the blocks completed we should be able to import them
		if reqRespData.req.RequestField(messages.RequestedDataHeader) {
			updatedFragment, ok := f.unreadyBlocks.updateDisjointFragments(reqRespData.responseData)
			if ok {
				validBlocks := validBlocksUnderFragment(highestFinalized.Number, updatedFragment)
				if len(validBlocks) > 0 {
					readyBlocks = append(readyBlocks, validBlocks)
				}
			} else {
				readyBlocks = append(readyBlocks, reqRespData.responseData)
			}

			continue
		}

		completedBlocks := f.unreadyBlocks.updateIncompleteBlocks(reqRespData.responseData)
		readyBlocks = append(readyBlocks, completedBlocks)
	}

	// disjoint fragments are pieces of the chain that could not be imported right now
	// because is blocks too far ahead or blocks that belongs to forks
	orderedFragments := sortFragmentsOfChain(readyBlocks)
	orderedFragments = mergeFragmentsOfChain(orderedFragments)

	nextBlocksToImport := make([]*types.BlockData, 0)
	disjointFragments := make([][]*types.BlockData, 0)

	for _, fragment := range orderedFragments {
		ok, err := f.blockState.HasHeader(fragment[0].Header.ParentHash)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return false, nil, nil, fmt.Errorf("checking block parent header: %w", err)
		}

		if ok {
			nextBlocksToImport = append(nextBlocksToImport, fragment...)
			continue
		}

		disjointFragments = append(disjointFragments, fragment)
	}

	logger.Debugf("blocks to import: %d, disjoint fragments: %d", len(nextBlocksToImport), len(disjointFragments))

	// this loop goal is to import ready blocks as well as
	// update the highestFinalized header
	for len(nextBlocksToImport) > 0 || len(disjointFragments) > 0 {
		for _, blockToImport := range nextBlocksToImport {
			imported, err := f.importer.handle(blockToImport, networkInitialSync)
			if err != nil {
				return false, nil, nil, fmt.Errorf("while handling ready block: %w", err)
			}

			if imported {
				f.syncedBlocks += 1
			}
		}

		nextBlocksToImport = make([]*types.BlockData, 0)
		highestFinalized, err = f.blockState.GetHighestFinalisedHeader()
		if err != nil {
			return false, nil, nil, fmt.Errorf("getting highest finalized header")
		}

		// check if blocks from the disjoint set can be imported on their on forks
		// given that fragment contains chains and these chains contains blocks
		// check if the first block in the chain contains a parent known by us
		for _, fragment := range disjointFragments {
			validFragment := validBlocksUnderFragment(highestFinalized.Number, fragment)
			if len(validFragment) == 0 {
				continue
			}

			ok, err := f.blockState.HasHeader(validFragment[0].Header.ParentHash)
			if err != nil && !errors.Is(err, database.ErrNotFound) {
				return false, nil, nil, err
			}

			if !ok {
				// if the parent of this valid fragment is behind our latest finalized number
				// then we can discard the whole fragment since it is a invalid fork
				if (validFragment[0].Header.Number - 1) <= highestFinalized.Number {
					continue
				}

				logger.Infof("starting an acestor search from %s parent of #%d (%s)",
					validFragment[0].Header.ParentHash,
					validFragment[0].Header.Number,
					validFragment[0].Header.Hash(),
				)

				f.unreadyBlocks.newDisjointFragemnt(validFragment)
				request := messages.NewBlockRequest(
					*variadic.Uint32OrHashFrom(validFragment[0].Header.ParentHash),
					messages.MaxBlocksInResponse,
					messages.BootstrapRequestData, messages.Descending)
				f.requestQueue.PushBack(request)
			} else {
				// inserting them in the queue to be processed after the main chain
				nextBlocksToImport = append(nextBlocksToImport, validFragment...)
			}
		}

		disjointFragments = nil
	}

	f.unreadyBlocks.removeIrrelevantFragments(highestFinalized.Number)
	return false, repChanges, peersToIgnore, nil
}

func (f *FullSyncStrategy) ShowMetrics() {
	totalSyncAndImportSeconds := time.Since(f.startedAt).Seconds()
	bps := float64(f.syncedBlocks) / totalSyncAndImportSeconds
	logger.Infof("⛓️ synced %d blocks, disjoint fragments %d, incomplete blocks %d, "+
		"took: %.2f seconds, bps: %.2f blocks/second, target block number #%d",
		f.syncedBlocks, len(f.unreadyBlocks.disjointFragments), len(f.unreadyBlocks.incompleteBlocks),
		totalSyncAndImportSeconds, bps, f.peers.getTarget())
}

func (f *FullSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	f.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

func (f *FullSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (
	gossip bool, repChange *Change, err error) {
	if f.blockState.IsPaused() {
		return false, nil, errors.New("blockstate service is paused")
	}

	blockAnnounceHeader := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	blockAnnounceHeaderHash := blockAnnounceHeader.Hash()

	logger.Infof("received block announce from %s: #%d (%s) best block: %v",
		from,
		blockAnnounceHeader.Number,
		blockAnnounceHeaderHash,
		msg.BestBlock,
	)

	if slices.Contains(f.badBlocks, blockAnnounceHeaderHash.String()) {
		logger.Infof("bad block receive from %s: #%d (%s) is a bad block",
			from, blockAnnounceHeader.Number, blockAnnounceHeaderHash)

		return false, &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			},
		}, nil
	}

	if msg.BestBlock {
		f.peers.update(from, blockAnnounceHeaderHash, uint32(blockAnnounceHeader.Number))
	}

	highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return false, nil, fmt.Errorf("get highest finalised header: %w", err)
	}

	// check if the announced block is relevant
	if blockAnnounceHeader.Number <= highestFinalized.Number || f.blockAlreadyTracked(blockAnnounceHeader) {
		logger.Infof("announced block irrelevant #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
		repChange = &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.NotRelevantBlockAnnounceValue,
				Reason: peerset.NotRelevantBlockAnnounceReason,
			},
		}

		return false, repChange, fmt.Errorf("%w: peer %s, block number #%d (%s)",
			errPeerOnInvalidFork, from, blockAnnounceHeader.Number, blockAnnounceHeaderHash.String())
	}

	logger.Infof("relevant announced block #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
	bestBlockHeader, err := f.blockState.BestBlockHeader()
	if err != nil {
		return false, nil, fmt.Errorf("get best block header: %w", err)
	}

	// if we still far from aproaching the calculated target
	// then we can ignore the block announce
	ratioOfCompleteness := (bestBlockHeader.Number / uint(f.peers.getTarget())) * 100
	logger.Infof("sync: ratio of completeness: %d", ratioOfCompleteness)
	if ratioOfCompleteness < 80 {
		return true, nil, nil
	}

	has, err := f.blockState.HasHeader(blockAnnounceHeaderHash)
	if err != nil {
		return false, nil, fmt.Errorf("checking if header exists: %w", err)
	}

	if !has {
		f.unreadyBlocks.newIncompleteBlock(blockAnnounceHeader)
		logger.Infof("requesting announced block body #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
		request := messages.NewBlockRequest(*variadic.Uint32OrHashFrom(blockAnnounceHeaderHash),
			1, messages.RequestedDataBody+messages.RequestedDataJustification, messages.Ascending)
		f.requestQueue.PushBack(request)
	}

	logger.Infof("announced block already exists #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
	return true, &Change{
		who: from,
		rep: peerset.ReputationChange{
			Value:  peerset.NotRelevantBlockAnnounceValue,
			Reason: peerset.NotRelevantBlockAnnounceReason,
		},
	}, nil
}

func (f *FullSyncStrategy) blockAlreadyTracked(announcedHeader *types.Header) bool {
	return f.unreadyBlocks.isIncomplete(announcedHeader.Hash()) ||
		f.unreadyBlocks.inDisjointFragment(announcedHeader.Hash(), announcedHeader.Number)
}

func (f *FullSyncStrategy) IsSynced() bool {
	highestBlock, err := f.blockState.BestBlockNumber()
	if err != nil {
		logger.Criticalf("cannot get best block number")
		return false
	}

	// TODO: research a better rule
	return uint32(highestBlock) >= (f.peers.getTarget() - 128)
}

type RequestResponseData struct {
	req          *messages.BlockRequestMessage
	responseData []*types.BlockData
}

func validateResults(results []*syncTaskResult, badBlocks []string) (repChanges []Change,
	peersToBlock []peer.ID, validRes []RequestResponseData) {

	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)
	validRes = make([]RequestResponseData, 0, len(results))

resultLoop:
	for _, result := range results {
		request := result.request.(*messages.BlockRequestMessage)

		if !result.completed {
			continue
		}

		response := result.response.(*messages.BlockResponseMessage)
		if request.Direction == messages.Descending {
			// reverse blocks before pre-validating and placing in ready queue
			slices.Reverse(response.BlockData)
		}

		err := validateResponseFields(request, response.BlockData)
		if err != nil {
			logger.Warnf("validating fields: %s", err)
			// TODO: check the reputation change for nil body in response
			// and nil justification in response
			if errors.Is(err, errNilHeaderInResponse) {
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.IncompleteHeaderValue,
						Reason: peerset.IncompleteHeaderReason,
					},
				})
			}

			continue
		}

		// only check if the responses forms a chain if the response contains the headers
		// of each block, othewise the response might only have the body/justification for
		// a block
		if request.RequestField(messages.RequestedDataHeader) && !isResponseAChain(response.BlockData) {
			logger.Warnf("response from %s is not a chain", result.who)
			repChanges = append(repChanges, Change{
				who: result.who,
				rep: peerset.ReputationChange{
					Value:  peerset.IncompleteHeaderValue,
					Reason: peerset.IncompleteHeaderReason,
				},
			})
			continue
		}

		for _, block := range response.BlockData {
			if slices.Contains(badBlocks, block.Hash.String()) {
				logger.Warnf("%s sent a known bad block: #%d (%s)",
					result.who, block.Number(), block.Hash.String())

				peersToBlock = append(peersToBlock, result.who)
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadBlockAnnouncementValue,
						Reason: peerset.BadBlockAnnouncementReason,
					},
				})

				continue resultLoop
			}
		}

		validRes = append(validRes, RequestResponseData{
			req:          request,
			responseData: response.BlockData,
		})
	}

	return repChanges, peersToBlock, validRes
}

// sortFragmentsOfChain will organise the fragments
// in a way we can import the older blocks first also guaranting that
// forks can be imported by organising them to be after the main chain
//
// e.g: consider the following fragment of chains
// [ {17} {1, 2, 3, 4, 5} {6, 7, 8, 9, 10} {8} {11, 12, 13, 14, 15, 16} ]
//
// note that we have fragments with single blocks, fragments with fork (in case of 8)
// after sorting these fragments we end up with:
// [ {1, 2, 3, 4, 5}  {6, 7, 8, 9, 10}  {8}  {11, 12, 13, 14, 15, 16}  {17} ]
func sortFragmentsOfChain(fragments [][]*types.BlockData) [][]*types.BlockData {
	if len(fragments) == 0 {
		return nil
	}

	slices.SortFunc(fragments, func(a, b []*types.BlockData) int {
		if a[0].Header.Number < b[0].Header.Number {
			return -1
		}
		if a[0].Header.Number == b[0].Header.Number {
			return 0
		}
		return 1
	})

	return fragments
}

// mergeFragmentsOfChain merges a sorted slice of fragments that forms a valid
// chain sequente which is the previous is the direct parent of the next block,
// and keep untouch fragments that does not forms such sequence,
// take as an example the following sorted slice.
// [ {1, 2, 3, 4, 5}  {6, 7, 8, 9, 10}  {8}  {11, 12, 13, 14, 15, 16}  {17} ]
// merge will transform it in the following slice:
// [ {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17} {8} ]
func mergeFragmentsOfChain(fragments [][]*types.BlockData) [][]*types.BlockData {
	if len(fragments) == 0 {
		return nil
	}

	mergedFragments := [][]*types.BlockData{fragments[0]}
	for i := 1; i < len(fragments); i++ {
		lastMerged := mergedFragments[len(mergedFragments)-1]
		current := fragments[i]

		if formsSequence(lastMerged[len(lastMerged)-1], current[0]) {
			mergedFragments[len(mergedFragments)-1] = append(lastMerged, current...)
		} else {
			mergedFragments = append(mergedFragments, current)
		}
	}

	return mergedFragments
}

// validBlocksUnderFragment ignore all blocks prior to the given last finalized number
func validBlocksUnderFragment(highestFinalizedNumber uint, fragmentBlocks []*types.BlockData) []*types.BlockData {
	startFragmentFrom := -1
	for idx, block := range fragmentBlocks {
		if block.Header.Number > highestFinalizedNumber {
			startFragmentFrom = idx
			break
		}
	}

	if startFragmentFrom < 0 {
		return nil
	}

	return fragmentBlocks[startFragmentFrom:]
}

// formsSequence given two fragments of blocks, check if they forms a sequence
// by comparing the latest block from the prev fragment with the
// first block of the next fragment
func formsSequence(prev, next *types.BlockData) bool {
	incrementOne := (prev.Header.Number + 1) == next.Header.Number
	isParent := prev.Hash == next.Header.ParentHash

	return incrementOne && isParent
}

// validateResponseFields checks that the expected fields are in the block data
func validateResponseFields(req *messages.BlockRequestMessage, blocks []*types.BlockData) error {
	for _, bd := range blocks {
		if req.RequestField(messages.RequestedDataHeader) && bd.Header == nil {
			return fmt.Errorf("%w: %s", errNilHeaderInResponse, bd.Hash)
		}

		if req.RequestField(messages.RequestedDataBody) && bd.Body == nil {
			return fmt.Errorf("%w: %s", errNilBodyInResponse, bd.Hash)
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
