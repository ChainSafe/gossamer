package sync

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
)

const blockRequestTimeout = 30 * time.Second

var _ Strategy = (*FullSyncStrategy)(nil)

var (
	errNilBlockData               = errors.New("block data is nil")
	errNilHeaderInResponse        = errors.New("expected header, received none")
	errNilBodyInResponse          = errors.New("expected body, received none")
	errNilJustificationInResponse = errors.New("expected justification, received none")
)

type FullSyncStrategy struct {
	bestBlockHeader *types.Header
	peers           *peerViewSet
	reqMaker        network.RequestMaker
	stopCh          chan struct{}
}

func NewFullSyncStrategy(startHeader *types.Header, reqMaker network.RequestMaker) *FullSyncStrategy {
	return &FullSyncStrategy{
		bestBlockHeader: startHeader,
		reqMaker:        reqMaker,
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
	currentTarget := f.peers.getTarget()
	// our best block is equal or ahead of current target
	// we're not legging behind, so let's set the set of
	// incomplete blocks and request them
	if uint32(f.bestBlockHeader.Number) >= currentTarget {
		return f.incompleteBlocksSync()
	}

	startRequestAt := f.bestBlockHeader.Number + 1
	targetBlockNumber := startRequestAt + uint(f.peers.len())*128

	if targetBlockNumber > uint(currentTarget) {
		targetBlockNumber = uint(currentTarget)
	}

	requests := network.NewAscendingBlockRequests(startRequestAt, targetBlockNumber,
		network.BootstrapRequestData)

	tasks := make([]*syncTask, len(requests))
	for idx, req := range requests {
		tasks[idx] = &syncTask{
			request:      req,
			response:     &network.BlockResponseMessage{},
			requestMaker: f.reqMaker,
		}
	}

	return tasks, nil
}

func (*FullSyncStrategy) IsFinished(results []*syncTaskResult) (bool, []Change, []peer.ID, error) {
	return false, nil, nil, nil
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

// handleWorkersResults, every time we submit requests to workers they results should be computed here
// and every cicle we should endup with a complete chain, whenever we identify
// any error from a worker we should evaluate the error and re-insert the request
// in the queue and wait for it to completes
// TODO: handle only justification requests
func (cs *FullSyncStrategy) handleWorkersResults(results []*syncTaskResult, origin BlockOrigin) error {
	repChanges := make([]Change, 0)
	blocks := make([]peer.ID, 0)

	for _, result := range results {
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
			continue
		}

		request := result.request.(*network.BlockRequestMessage)
		response := result.response.(*network.BlockResponseMessage)

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
				repChanges = append(repChanges, Change{
					who: result.who,
					rep: peerset.ReputationChange{
						Value:  peerset.IncompleteHeaderValue,
						Reason: peerset.IncompleteHeaderReason,
					},
				})
			}

			err = cs.submitRequest(taskResult.request, nil, workersResults)
			if err != nil {
				return err
			}
			continue taskResultLoop
		}
	}

taskResultLoop:
	for waitingBlocks > 0 {
		// in a case where we don't handle workers results we should check the pool
		idleDuration := time.Minute
		idleTimer := time.NewTimer(idleDuration)

		select {
		case <-cs.stopCh:
			return nil

		case <-idleTimer.C:
			return ErrResultsTimeout

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
				if !errors.Is(taskResult.err, network.ErrReceivedEmptyMessage) {
					cs.workerPool.ignorePeerAsWorker(taskResult.who)

					logger.Errorf("task result: peer(%s) error: %s",
						taskResult.who, taskResult.err)

					if strings.Contains(taskResult.err.Error(), "protocols not supported") {
						cs.network.ReportPeer(peerset.ReputationChange{
							Value:  peerset.BadProtocolValue,
							Reason: peerset.BadProtocolReason,
						}, who)
					}

					if errors.Is(taskResult.err, network.ErrNilBlockInResponse) {
						cs.network.ReportPeer(peerset.ReputationChange{
							Value:  peerset.BadMessageValue,
							Reason: peerset.BadMessageReason,
						}, who)
					}
				}

				// TODO: avoid the same peer to get the same task
				err := cs.submitRequest(request, nil, workersResults)
				if err != nil {
					return err
				}
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

				err = cs.submitRequest(taskResult.request, nil, workersResults)
				if err != nil {
					return err
				}
				continue taskResultLoop
			}

			isChain := isResponseAChain(response.BlockData)
			if !isChain {
				logger.Criticalf("response from %s is not a chain", who)
				err = cs.submitRequest(taskResult.request, nil, workersResults)
				if err != nil {
					return err
				}
				continue taskResultLoop
			}

			grows := doResponseGrowsTheChain(response.BlockData, syncingChain,
				startAtBlock, expectedSyncedBlocks)
			if !grows {
				logger.Criticalf("response from %s does not grows the ongoing chain", who)
				err = cs.submitRequest(taskResult.request, nil, workersResults)
				if err != nil {
					return err
				}
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
					err = cs.submitRequest(taskResult.request, nil, workersResults)
					if err != nil {
						return err
					}
					continue taskResultLoop
				}

				blockExactIndex := blockInResponse.Header.Number - startAtBlock
				if blockExactIndex < uint(expectedSyncedBlocks) {
					syncingChain[blockExactIndex] = blockInResponse
				}
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
				err = cs.submitRequest(taskResult.request, nil, workersResults)
				if err != nil {
					return err
				}
				continue taskResultLoop
			}
		}
	}

	retreiveBlocksSeconds := time.Since(startTime).Seconds()
	logger.Infof("🔽 retrieved %d blocks, took: %.2f seconds, starting process...",
		expectedSyncedBlocks, retreiveBlocksSeconds)

	// response was validated! place into ready block queue
	for _, bd := range syncingChain {
		// block is ready to be processed!
		if err := cs.handleReadyBlock(bd, origin); err != nil {
			return fmt.Errorf("while handling ready block: %w", err)
		}
	}

	cs.showSyncStats(startTime, len(syncingChain))
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

func reverseBlockData(data []*types.BlockData) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
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
	if firstBlockExactIndex != 0 && firstBlockExactIndex < uint(expectedTotal) {
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
