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

	"github.com/libp2p/go-libp2p/core/peer"
)

const defaultNumOfTasks = 3

var _ Strategy = (*FullSyncStrategy)(nil)

var (
	errFailedToGetParent   = errors.New("failed to get parent header")
	errNilHeaderInResponse = errors.New("expected header, received none")
	errNilBodyInResponse   = errors.New("expected body, received none")
	errBadBlockReceived    = errors.New("bad block received")
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

type importer interface {
	importBlock(*types.BlockData, BlockOrigin) (imported bool, err error)
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
	blockImporter importer
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
		blockImporter: newBlockImporter(cfg),
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

func (f *FullSyncStrategy) NextActions() ([]*SyncTask, error) {
	f.startedAt = time.Now()
	f.syncedBlocks = 0

	var reqsFromQueue []*messages.BlockRequestMessage

	for i := 0; i < f.numOfTasks; i++ {
		msg, ok := f.requestQueue.PopFront()
		if !ok {
			break
		}

		reqsFromQueue = append(reqsFromQueue, msg)
	}

	currentTarget := f.peers.getTarget()
	bestBlockHeader, err := f.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("while getting best block header")
	}

	// our best block is equal or ahead of current target.
	// in the node's pov we are not lagging behind so there's nothing to do
	// or we didn't receive block announces, so lets ask for more blocks
	if uint32(bestBlockHeader.Number) >= currentTarget { //nolint:gosec
		return f.createTasks(reqsFromQueue), nil
	}

	startRequestAt := bestBlockHeader.Number + 1
	targetBlockNumber := startRequestAt + uint(f.numOfTasks)*127 //nolint:gosec

	if targetBlockNumber > uint(currentTarget) {
		targetBlockNumber = uint(currentTarget)
	}

	ascendingBlockRequests := messages.NewAscendingBlockRequests(
		startRequestAt, targetBlockNumber,
		messages.BootstrapRequestData)
	reqsFromQueue = append(reqsFromQueue, ascendingBlockRequests...)

	return f.createTasks(reqsFromQueue), nil
}

func (f *FullSyncStrategy) createTasks(requests []*messages.BlockRequestMessage) []*SyncTask {
	tasks := make([]*SyncTask, 0, len(requests))
	for _, req := range requests {
		tasks = append(tasks, &SyncTask{
			request:      req,
			response:     &messages.BlockResponseMessage{},
			requestMaker: f.reqMaker,
		})
	}
	return tasks
}

// Process receives as arguments the peer-to-peer block request responses
// and will check if the blocks data in the response can be imported to the state
// or complete an incomplete block or is part of a disjoint block set which will
// as a result it returns the if the strategy is finished, the peer reputations to change,
// peers to block/ban, or an error. FullSyncStrategy is intended to run as long as the node lives.
func (f *FullSyncStrategy) Process(results []*SyncTaskResult) (
	isFinished bool, reputations []Change, bans []peer.ID, err error) {
	repChanges, peersToIgnore, validResp := validateResults(results, f.badBlocks)
	logger.Debugf("evaluating %d task results, %d valid responses", len(results), len(validResp))

	var highestFinalized *types.Header
	highestFinalized, err = f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return false, nil, nil, fmt.Errorf("getting highest finalized header")
	}

	readyBlocks := make([][]*types.BlockData, 0, len(validResp))
	for _, reqRespData := range validResp {
		// if Gossamer requested the header, then the response data should contains
		// the full blocks to be imported. If Gossamer didn't request the header,
		// then the response should only contain the missing parts that will complete
		// the unreadyBlocks and then with the blocks completed we should be able to import them
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
	sortFragmentsOfChain(readyBlocks)
	orderedFragments := mergeFragmentsOfChain(readyBlocks)

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

	// this loop goal is to import ready blocks as well as update the highestFinalized header
	for len(nextBlocksToImport) > 0 || len(disjointFragments) > 0 {
		for _, blockToImport := range nextBlocksToImport {
			imported, err := f.blockImporter.importBlock(blockToImport, networkInitialSync)
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

				f.unreadyBlocks.newDisjointFragment(validFragment)
				request := messages.NewBlockRequest(
					*messages.NewFromBlock(validFragment[0].Header.ParentHash),
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
	logger.Infof("⛓️ synced %d blocks, tasks on queue %d, disjoint fragments %d, incomplete blocks %d, "+
		"took: %.2f seconds, bps: %.2f blocks/second, target block number #%d",
		f.syncedBlocks, f.requestQueue.Len(), len(f.unreadyBlocks.disjointFragments), len(f.unreadyBlocks.incompleteBlocks),
		totalSyncAndImportSeconds, bps, f.peers.getTarget())
}

func (f *FullSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	f.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

func (f *FullSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (
	repChange *Change, err error) {
	if f.blockState.IsPaused() {
		return nil, errors.New("blockstate service is paused")
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
		logger.Infof("bad block received from %s: #%d (%s) is a bad block",
			from, blockAnnounceHeader.Number, blockAnnounceHeaderHash)

		return &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			},
		}, errBadBlockReceived
	}

	if msg.BestBlock {
		f.peers.update(from, blockAnnounceHeaderHash, uint32(blockAnnounceHeader.Number)) //nolint:gosec
	}

	highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, fmt.Errorf("get highest finalised header: %w", err)
	}

	// check if the announced block is relevant
	if blockAnnounceHeader.Number <= highestFinalized.Number || f.blockAlreadyTracked(blockAnnounceHeader) {
		logger.Infof("ignoring announced block #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
		repChange = &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.NotRelevantBlockAnnounceValue,
				Reason: peerset.NotRelevantBlockAnnounceReason,
			},
		}

		return repChange, nil
	}

	logger.Infof("relevant announced block #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
	bestBlockHeader, err := f.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("get best block header: %w", err)
	}

	// if we still far from aproaching the announced block
	// then we can ignore the block announce
	mx := max(blockAnnounceHeader.Number, bestBlockHeader.Number)
	mn := min(blockAnnounceHeader.Number, bestBlockHeader.Number)
	if (mx - mn) > messages.MaxBlocksInResponse {
		return nil, nil
	}

	has, err := f.blockState.HasHeader(blockAnnounceHeaderHash)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, fmt.Errorf("checking if header exists: %w", err)
		}
	}

	if !has {
		f.unreadyBlocks.newIncompleteBlock(blockAnnounceHeader)
		logger.Infof("requesting announced block body #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
		request := messages.NewBlockRequest(*messages.NewFromBlock(blockAnnounceHeaderHash),
			1, messages.RequestedDataBody+messages.RequestedDataJustification, messages.Ascending)
		f.requestQueue.PushBack(request)
	} else {
		logger.Infof("announced block already exists #%d (%s)", blockAnnounceHeader.Number, blockAnnounceHeaderHash.Short())
	}

	return &Change{
		who: from,
		rep: peerset.ReputationChange{
			Value:  peerset.GossipSuccessValue,
			Reason: peerset.GossipSuccessReason,
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

	logger.Infof("highest block: %d target %d", highestBlock, f.peers.getTarget())
	return uint32(highestBlock)+messages.MaxBlocksInResponse >= f.peers.getTarget() //nolint:gosec
}

func (f *FullSyncStrategy) Result() any {
	logger.Debug("trying to get a result from full sync strategy which is supposed to run forever")
	return nil
}

type RequestResponseData struct {
	req          *messages.BlockRequestMessage
	responseData []*types.BlockData
}

func validateResults(results []*SyncTaskResult, badBlocks []string) (repChanges []Change,
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
func sortFragmentsOfChain(fragments [][]*types.BlockData) {
	if len(fragments) == 0 {
		return
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
}

// mergeFragmentsOfChain expects a sorted slice of fragments and merges those
// fragments for which the last block of the previous fragment is the direct parent of
// the first block of the next fragment.
// Fragments that are not part of this sequence (e.g. from forks) are left untouched.
// Take as an example the following sorted slice:
// [ {1, 2, 3, 4, 5}  {6, 7, 8, 9, 10}  {8}  {11, 12, 13, 14, 15, 16}  {17} ]
// merge will transform it to the following slice:
// [ {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17} {8} ]
func mergeFragmentsOfChain(fragments [][]*types.BlockData) [][]*types.BlockData {
	if len(fragments) == 0 {
		return nil
	}

	mergedFragments := [][]*types.BlockData{fragments[0]}
	for i := 1; i < len(fragments); i++ {
		lastMergedFragment := mergedFragments[len(mergedFragments)-1]
		currentFragment := fragments[i]

		lastBlock := lastMergedFragment[len(lastMergedFragment)-1]

		if lastBlock.IsParent(currentFragment[0]) {
			mergedFragments[len(mergedFragments)-1] = append(lastMergedFragment, currentFragment...)
		} else {
			mergedFragments = append(mergedFragments, currentFragment)
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
		if !previousBlockData.IsParent(currBlockData) {
			return false
		}

		previousBlockData = currBlockData
	}

	return true
}
