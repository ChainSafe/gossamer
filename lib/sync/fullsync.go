package sync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var _ Strategy = (*FullSyncStrategy)(nil)

var (
	errFailedToGetParent          = errors.New("failed to get parent header")
	errNilBlockData               = errors.New("block data is nil")
	errNilHeaderInResponse        = errors.New("expected header, received none")
	errNilBodyInResponse          = errors.New("expected body, received none")
	errNilJustificationInResponse = errors.New("expected justification, received none")

	blockSizeGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_sync",
		Name:      "block_size",
		Help:      "represent the size of blocks synced",
	})
)

type (
	// Telemetry is the telemetry client to send telemetry messages.
	Telemetry interface {
		SendMessage(msg json.Marshaler)
	}

	// StorageState is the interface for the storage state
	StorageState interface {
		TrieState(root *common.Hash) (*rtstorage.TrieState, error)
		sync.Locker
	}

	// TransactionState is the interface for transaction queue methods
	TransactionState interface {
		RemoveExtrinsic(ext types.Extrinsic)
	}

	// BabeVerifier deals with BABE block verification
	BabeVerifier interface {
		VerifyBlock(header *types.Header) error
	}

	// FinalityGadget implements justification verification functionality
	FinalityGadget interface {
		VerifyBlockJustification(common.Hash, []byte) error
	}

	// BlockImportHandler is the interface for the handler of newly imported blocks
	BlockImportHandler interface {
		HandleBlockImport(block *types.Block, state *rtstorage.TrieState, announce bool) error
	}
)

// Config is the configuration for the sync Service.
type FullSyncConfig struct {
	StartHeader        *types.Header
	BlockState         BlockState
	StorageState       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BlockImportHandler BlockImportHandler
	BabeVerifier       BabeVerifier
	Telemetry          Telemetry
	BadBlocks          []string
	RequestMaker       network.RequestMaker
}

type FullSyncStrategy struct {
	bestBlockHeader    *types.Header
	missingRequests    []*network.BlockRequestMessage
	disjointBlocks     [][]*types.BlockData
	peers              *peerViewSet
	badBlocks          []string
	reqMaker           network.RequestMaker
	blockState         BlockState
	storageState       StorageState
	transactionState   TransactionState
	babeVerifier       BabeVerifier
	finalityGadget     FinalityGadget
	blockImportHandler BlockImportHandler
	telemetry          Telemetry

	startedAt    time.Time
	syncedBlocks int
}

func NewFullSyncStrategy(cfg *FullSyncConfig) *FullSyncStrategy {
	return &FullSyncStrategy{
		badBlocks:          cfg.BadBlocks,
		bestBlockHeader:    cfg.StartHeader,
		reqMaker:           cfg.RequestMaker,
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		transactionState:   cfg.TransactionState,
		babeVerifier:       cfg.BabeVerifier,
		finalityGadget:     cfg.FinalityGadget,
		blockImportHandler: cfg.BlockImportHandler,
		telemetry:          cfg.Telemetry,
		peers: &peerViewSet{
			view:   make(map[peer.ID]peerView),
			target: 0,
		},
	}
}

func (f *FullSyncStrategy) incompleteBlocksSync() ([]*syncTask, error) {
	panic("incompleteBlocksSync not implemented yet")
}

func (f *FullSyncStrategy) NextActions() ([]*syncTask, error) {
	f.startedAt = time.Now()

	if len(f.missingRequests) > 0 {
		return f.createTasks(f.missingRequests), nil
	}

	currentTarget := f.peers.getTarget()
	// our best block is equal or ahead of current target
	// we're not legging behind, so let's set the set of
	// incomplete blocks and request them
	if uint32(f.bestBlockHeader.Number) >= currentTarget {
		return f.incompleteBlocksSync()
	}

	startRequestAt := f.bestBlockHeader.Number + 1
	targetBlockNumber := startRequestAt + 60*128

	if targetBlockNumber > uint(currentTarget) {
		targetBlockNumber = uint(currentTarget)
	}

	requests := network.NewAscendingBlockRequests(startRequestAt, targetBlockNumber,
		network.BootstrapRequestData)
	return f.createTasks(requests), nil
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
	repChanges, blocks, missingReq, validResp := validateResults(results, f.badBlocks)
	f.missingRequests = missingReq

	if f.disjointBlocks == nil {
		f.disjointBlocks = make([][]*types.BlockData, 0)
	}

	// merge validResp with the current disjoint blocks
	for _, resp := range validResp {
		f.disjointBlocks = append(f.disjointBlocks, resp.BlockData)
	}

	// given the validResponses, can we start importing the blocks or
	// we should wait for the missing requests to fill the gap?
	blocksToImport, disjointBlocks := blocksAvailable(f.bestBlockHeader.Hash(), f.bestBlockHeader.Number, f.disjointBlocks)
	f.disjointBlocks = disjointBlocks

	if len(blocksToImport) > 0 {
		for _, blockToImport := range blocksToImport {
			err := f.handleReadyBlock(blockToImport, networkInitialSync)
			if err != nil {
				return false, nil, nil, fmt.Errorf("while handling ready block: %w", err)
			}
			f.bestBlockHeader = blockToImport.Header
		}
	}

	f.syncedBlocks = len(blocksToImport)
	return false, repChanges, blocks, nil
}

func (f *FullSyncStrategy) ShowMetrics() {
	totalSyncAndImportSeconds := time.Since(f.startedAt).Seconds()
	bps := float64(f.syncedBlocks) / totalSyncAndImportSeconds
	logger.Infof("⛓️ synced %d blocks, "+
		"took: %.2f seconds, bps: %.2f blocks/second, target block number #%d",
		f.syncedBlocks, totalSyncAndImportSeconds, bps, f.peers.getTarget())
}

func (f *FullSyncStrategy) OnBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	logger.Infof("received block announce from %s: #%d (%s)",
		from,
		msg.BestBlockNumber,
		msg.BestBlockHash.Short(),
	)

	f.peers.update(from, msg.BestBlockHash, msg.BestBlockNumber)
	return nil
}

func (*FullSyncStrategy) OnBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	logger.Infof("received block announce: %d", msg.Number)
	return nil
}

var ErrResultsTimeout = errors.New("waiting results reached timeout")

func (f *FullSyncStrategy) handleReadyBlock(bd *types.BlockData, origin BlockOrigin) error {
	err := f.processBlockData(*bd, origin)
	if err != nil {
		// depending on the error, we might want to save this block for later
		logger.Errorf("processing block #%d (%s) failed: %s", bd.Header.Number, bd.Hash, err)
		return err
	}

	return nil
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
// TODO: https://github.com/ChainSafe/gossamer/issues/3468
func (f *FullSyncStrategy) processBlockData(blockData types.BlockData, origin BlockOrigin) error {
	// while in bootstrap mode we don't need to broadcast block announcements
	// TODO: set true if not in initial sync setup
	announceImportedBlock := false

	if blockData.Header != nil {
		if blockData.Body != nil {
			err := f.processBlockDataWithHeaderAndBody(blockData, origin, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			err := f.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err := f.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (f *FullSyncStrategy) processBlockDataWithHeaderAndBody(blockData types.BlockData,
	origin BlockOrigin, announceImportedBlock bool) (err error) {

	if origin != networkInitialSync {
		err = f.babeVerifier.VerifyBlock(blockData.Header)
		if err != nil {
			return fmt.Errorf("babe verifying block: %w", err)
		}
	}

	f.handleBody(blockData.Body)

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = f.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (f *FullSyncStrategy) handleBody(body *types.Body) {
	acc := 0
	for _, ext := range *body {
		acc += len(ext)
		f.transactionState.RemoveExtrinsic(ext)
	}

	blockSizeGauge.Set(float64(acc))
}

// handleHeader handles blocks (header+body) included in BlockResponses
func (f *FullSyncStrategy) handleBlock(block *types.Block, announceImportedBlock bool) error {
	parent, err := f.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	f.storageState.Lock()
	defer f.storageState.Unlock()

	ts, err := f.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	rt, err := f.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = f.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	blockHash := block.Header.Hash()
	f.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (f *FullSyncStrategy) handleJustification(header *types.Header, justification []byte) (err error) {
	headerHash := header.Hash()
	err = f.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = f.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	return nil
}

func validateResults(results []*syncTaskResult, badBlocks []string) (repChanges []Change, blocks []peer.ID,
	missingReqs []*network.BlockRequestMessage, validRes []*network.BlockResponseMessage) {
	repChanges = make([]Change, 0)
	blocks = make([]peer.ID, 0)

	missingReqs = make([]*network.BlockRequestMessage, 0, len(results))
	validRes = make([]*network.BlockResponseMessage, 0, len(results))

resultLoop:
	for _, result := range results {
		request := result.request.(*network.BlockRequestMessage)

		if result.err != nil {
			if !errors.Is(result.err, network.ErrReceivedEmptyMessage) {
				blocks = append(blocks, result.who)

				if strings.Contains(result.err.Error(), "protocols not supported") {
					repChanges = append(repChanges, Change{
						who: result.who,
						rep: peerset.ReputationChange{
							Value:  peerset.BadProtocolValue,
							Reason: peerset.BadProtocolReason,
						},
					})
				}

				if errors.Is(result.err, network.ErrNilBlockInResponse) {
					repChanges = append(repChanges, Change{
						who: result.who,
						rep: peerset.ReputationChange{
							Value:  peerset.BadMessageValue,
							Reason: peerset.BadMessageReason,
						},
					})
				}
			}

			missingReqs = append(missingReqs, request)
			continue
		}

		response := result.response.(*network.BlockResponseMessage)
		if request.Direction == network.Descending {
			// reverse blocks before pre-validating and placing in ready queue
			slices.Reverse(response.BlockData)
		}

		err := validateResponseFields(request.RequestedData, response.BlockData)
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

			missingReqs = append(missingReqs, request)
			continue
		}

		if !isResponseAChain(response.BlockData) {
			logger.Criticalf("response from %s is not a chain", result.who)
			missingReqs = append(missingReqs, request)
			continue
		}

		for _, block := range response.BlockData {
			if slices.Contains(badBlocks, block.Hash.String()) {
				logger.Criticalf("%s sent a known bad block: #%d (%s)",
					result.who, block.Number(), block.Hash.String())

				blocks = append(blocks, result.who)
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.BadBlockAnnouncementValue,
						Reason: peerset.BadBlockAnnouncementReason,
					},
				})

				missingReqs = append(missingReqs, request)
				continue resultLoop
			}
		}

		validRes = append(validRes, response)
	}

	return repChanges, blocks, missingReqs, validRes
}

// blocksAvailable given a set of responses, which are fragments of the chain we should
// check if there is fragments that can be imported or fragments that are disjoint (cannot be imported yet)
func blocksAvailable(blockHash common.Hash, blockNumber uint, responses [][]*types.BlockData) (
	[]*types.BlockData, [][]*types.BlockData) {
	if len(responses) == 0 {
		return nil, nil
	}

	slices.SortFunc(responses, func(a, b []*types.BlockData) int {
		if a[len(a)-1].Header.Number < b[0].Header.Number {
			return -1
		}
		if a[len(a)-1].Header.Number == b[0].Header.Number {
			return 0
		}
		return 1
	})

	type hashAndNumber struct {
		hash   common.Hash
		number uint
	}

	compareWith := hashAndNumber{
		hash:   blockHash,
		number: blockNumber,
	}

	disjoints := false
	lastIdx := 0

	okFrag := make([]*types.BlockData, 0, len(responses))
	for idx, chain := range responses {
		if len(chain) == 0 {
			panic("unreachable")
		}

		incrementOne := (compareWith.number + 1) == chain[0].Header.Number
		isParent := compareWith.hash == chain[0].Header.ParentHash

		fmt.Printf("checking: in response %d, compare with %d\n", chain[0].Header.Number, compareWith.number+1)
		fmt.Printf("checking: in response %s, compare with %s\n", chain[0].Header.ParentHash, compareWith.hash)

		if incrementOne && isParent {
			okFrag = append(okFrag, chain...)
			compareWith = hashAndNumber{
				hash:   chain[len(chain)-1].Hash,
				number: chain[len(chain)-1].Header.Number,
			}
			continue
		}

		lastIdx = idx
		disjoints = true
		break
	}

	if disjoints {
		return okFrag, responses[lastIdx:]
	}

	return okFrag, nil
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
