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
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const defaultNumOfTasks = 3

var _ Strategy = (*FullSyncStrategy)(nil)

var (
	errFailedToGetParent             = errors.New("failed to get parent header")
	errNilHeaderInResponse           = errors.New("expected header, received none")
	errNilBodyInResponse             = errors.New("expected body, received none")
	errPeerOnInvalidFork             = errors.New("peer is on an invalid fork")
	errMismatchBestBlockAnnouncement = errors.New("mismatch best block announcement")

	blockSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_sync",
		Name:      "block_size",
		Help:      "represent the size of blocks synced",
	})
)

// Config is the configuration for the sync Service.
type FullSyncConfig struct {
	StartHeader        *types.Header
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

type FullSyncStrategy struct {
	requestQueue  *requestsQueue[*network.BlockRequestMessage]
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
		badBlocks:  cfg.BadBlocks,
		reqMaker:   cfg.RequestMaker,
		blockState: cfg.BlockState,
		numOfTasks: cfg.NumOfTasks,
		importer:   newBlockImporter(cfg),
		unreadyBlocks: &unreadyBlocks{
			incompleteBlocks: make(map[common.Hash]*types.BlockData),
			// TODO: cap disjoitChains to don't grows indefinitely
			disjointChains: make([][]*types.BlockData, 0),
		},
		requestQueue: &requestsQueue[*network.BlockRequestMessage]{
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

	if f.requestQueue.Len() > 0 {
		message, _ := f.requestQueue.PopFront()
		return f.createTasks([]*network.BlockRequestMessage{message}), nil
	}

	currentTarget := f.peers.getTarget()
	bestBlockHeader, err := f.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("while getting best block header")
	}

	// our best block is equal or ahead of current target.
	// in the nodes pov we are not legging behind so there's nothing to do
	if uint32(bestBlockHeader.Number) >= currentTarget {
		return nil, nil
	}

	startRequestAt := bestBlockHeader.Number + 1
	targetBlockNumber := startRequestAt + 128

	if targetBlockNumber > uint(currentTarget) {
		targetBlockNumber = uint(currentTarget)
	}

	ascendingBlockRequests := network.NewAscendingBlockRequests(startRequestAt, targetBlockNumber,
		network.BootstrapRequestData)

	return f.createTasks(ascendingBlockRequests), nil
}

func (f *FullSyncStrategy) createTasks(requests []*network.BlockRequestMessage) []*syncTask {
	tasks := make([]*syncTask, len(requests))
	for idx, req := range requests {
		tasks[idx] = &syncTask{
			request:      req,
			response:     &network.BlockResponseMessage{},
			requestMaker: f.reqMaker,
		}
	}
	return tasks
}

func (f *FullSyncStrategy) IsFinished(results []*syncTaskResult) (bool, []Change, []peer.ID, error) {
	repChanges, peersToIgnore, validResp := validateResults(results, f.badBlocks)

	validBlocksUnderFragment := func(highestFinalizedNumber uint, fragmentBlocks []*types.BlockData) []*types.BlockData {
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

	highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return false, nil, nil, fmt.Errorf("getting highest finalized header")
	}

	readyBlocks := make([][]*types.BlockData, 0, len(validResp))
	for _, reqRespData := range validResp {
		// if Gossamer requested the header, then the response data should
		// contains the full bocks to be imported
		// if Gossamer don't requested the header, then the response shoul
		// only contains the missing parts the will complete the unreadyBlocks
		// and then with the blocks completed we should be able to import them

		if reqRespData.req.RequestField(network.RequestedDataHeader) {
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

	// disjoint fragments are pieces of the chain that could not be imported rn
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

		// check if blocks from the disjoint set can be imported on their on forks
		// given that fragment contains chains and these chains contains blocks
		// check if the first block in the chain contains a parent known by us
		for _, fragment := range disjointFragments {
			highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
			if err != nil {
				return false, nil, nil, fmt.Errorf("getting highest finalized header")
			}

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

				f.unreadyBlocks.newFragment(validFragment)
				request := network.NewBlockRequest(
					*variadic.FromHash(validFragment[0].Header.ParentHash),
					network.MaxBlocksInResponse,
					network.BootstrapRequestData, network.Descending)
				f.requestQueue.PushBack(request)
			} else {
				// inserting them in the queue to be processed after the main chain
				nextBlocksToImport = append(nextBlocksToImport, validFragment...)
			}
		}

		disjointFragments = nil
	}

	return false, repChanges, peersToIgnore, nil
}

func (f *FullSyncStrategy) ShowMetrics() {
	totalSyncAndImportSeconds := time.Since(f.startedAt).Seconds()
	bps := float64(f.syncedBlocks) / totalSyncAndImportSeconds
	logger.Infof("⛓️ synced %d blocks, disjoint fragments %d, incomplete blocks %d, "+
		"took: %.2f seconds, bps: %.2f blocks/second, target block number #%d",
		f.syncedBlocks, len(f.unreadyBlocks.disjointChains), len(f.unreadyBlocks.incompleteBlocks),
		totalSyncAndImportSeconds, bps, f.peers.getTarget())
}

func (f *FullSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	f.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

func (f *FullSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) (repChange *Change, err error) {
	if f.blockState.IsPaused() {
		return nil, errors.New("blockstate service is paused")
	}

	if msg.BestBlock {
		pv := f.peers.get(from)
		if uint(pv.bestBlockNumber) != msg.Number {
			repChange = &Change{
				who: from,
				rep: peerset.ReputationChange{
					Value:  peerset.BadBlockAnnouncementValue,
					Reason: peerset.BadBlockAnnouncementReason,
				},
			}
			return repChange, fmt.Errorf("%w: peer %s, on handshake #%d, on announce #%d",
				errMismatchBestBlockAnnouncement, from, pv.bestBlockNumber, msg.Number)
		}
	}

	currentTarget := f.peers.getTarget()
	if msg.Number >= uint(currentTarget) {
		return nil, nil
	}

	blockAnnounceHeader := types.NewHeader(msg.ParentHash, msg.StateRoot, msg.ExtrinsicsRoot, msg.Number, msg.Digest)
	blockAnnounceHeaderHash := blockAnnounceHeader.Hash()

	logger.Infof("received block announce from %s: #%d (%s) best block: %v",
		from,
		blockAnnounceHeader.Number,
		blockAnnounceHeaderHash,
		msg.BestBlock,
	)

	// check if their best block is on an invalid chain, if it is,
	// potentially downscore them for now, we can remove them from the syncing peers set
	highestFinalized, err := f.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, fmt.Errorf("get highest finalised header: %w", err)
	}

	if blockAnnounceHeader.Number <= highestFinalized.Number {
		repChange = &Change{
			who: from,
			rep: peerset.ReputationChange{
				Value:  peerset.BadBlockAnnouncementValue,
				Reason: peerset.BadBlockAnnouncementReason,
			},
		}
		return repChange, fmt.Errorf("%w: peer %s, block number #%d (%s)",
			errPeerOnInvalidFork, from, blockAnnounceHeader.Number, blockAnnounceHeaderHash.String())
	}

	has, err := f.blockState.HasHeader(blockAnnounceHeaderHash)
	if err != nil {
		return nil, fmt.Errorf("checking if header exists: %w", err)
	}

	if !has {
		f.unreadyBlocks.newHeader(blockAnnounceHeader)
		request := network.NewBlockRequest(*variadic.FromHash(blockAnnounceHeaderHash),
			1, network.RequestedDataBody+network.RequestedDataJustification, network.Ascending)
		f.requestQueue.PushBack(request)
	}

	return nil, nil
}

type RequestResponseData struct {
	req          *network.BlockRequestMessage
	responseData []*types.BlockData
}

func validateResults(results []*syncTaskResult, badBlocks []string) (repChanges []Change,
	peersToBlock []peer.ID, validRes []RequestResponseData) {

	repChanges = make([]Change, 0)
	peersToBlock = make([]peer.ID, 0)
	validRes = make([]RequestResponseData, 0, len(results))

resultLoop:
	for _, result := range results {
		request := result.request.(*network.BlockRequestMessage)

		if !result.completed {
			continue
		}

		response := result.response.(*network.BlockResponseMessage)
		if request.Direction == network.Descending {
			// reverse blocks before pre-validating and placing in ready queue
			slices.Reverse(response.BlockData)
		}

		err := validateResponseFields(request, response.BlockData)
		if err != nil {
			logger.Criticalf("validating fields: %s", err)
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

			//missingReqs = append(missingReqs, request)
			continue
		}

		// only check if the responses forms a chain if the response contains the headers
		// of each block, othewise the response might only have the body/justification for
		// a block
		if request.RequestField(network.RequestedDataHeader) && !isResponseAChain(response.BlockData) {
			logger.Criticalf("response from %s is not a chain", result.who)
			repChanges = append(repChanges, Change{
				who: result.who,
				rep: peerset.ReputationChange{
					Value:  peerset.IncompleteHeaderValue,
					Reason: peerset.IncompleteHeaderReason,
				},
			})
			//missingReqs = append(missingReqs, request)
			continue
		}

		for _, block := range response.BlockData {
			if slices.Contains(badBlocks, block.Hash.String()) {
				logger.Criticalf("%s sent a known bad block: #%d (%s)",
					result.who, block.Number(), block.Hash.String())

				peersToBlock = append(peersToBlock, result.who)
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadBlockAnnouncementValue,
						Reason: peerset.BadBlockAnnouncementReason,
					},
				})

				//missingReqs = append(missingReqs, request)
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
func sortFragmentsOfChain(responses [][]*types.BlockData) [][]*types.BlockData {
	if len(responses) == 0 {
		return nil
	}

	slices.SortFunc(responses, func(a, b []*types.BlockData) int {
		if a[0].Header.Number < b[0].Header.Number {
			return -1
		}
		if a[0].Header.Number == b[0].Header.Number {
			return 0
		}
		return 1
	})

	return responses
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

func formsSequence(last, curr *types.BlockData) bool {
	incrementOne := (last.Header.Number + 1) == curr.Header.Number
	isParent := last.Hash == curr.Header.ParentHash

	return incrementOne && isParent
}

// validateResponseFields checks that the expected fields are in the block data
func validateResponseFields(req *network.BlockRequestMessage, blocks []*types.BlockData) error {
	for _, bd := range blocks {
		if req.RequestField(network.RequestedDataHeader) && bd.Header == nil {
			return fmt.Errorf("%w: %s", errNilHeaderInResponse, bd.Hash)
		}

		if req.RequestField(network.RequestedDataBody) && bd.Body == nil {
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
