package sync

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	maxBlocksToRequest = 128
	maxImportDequeLen  = 1024
	maxRunningWorkers  = 50
)

var bootstrapRequestData = network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification

type bootstrapSyncer struct {
	blockState         BlockState
	storageState       StorageState
	network            Network
	blockImportHandler BlockImportHandler
	babeVerifier       BabeVerifier
	transactionState   TransactionState
	telemetry          Telemetry
	finalityGadget     FinalityGadget

	offset uint
}

func (b *bootstrapSyncer) Sync() {
	for {
		availablePeers := b.network.TotalConnectedPeers()
		for len(availablePeers) < 1 {
			// TODO: wait for connections
			time.Sleep(10 * time.Second)
			availablePeers = b.network.TotalConnectedPeers()
		}

		workers := make(map[peer.ID]*syncerWorkerByNumber, len(availablePeers))
		for _, availablePeer := range availablePeers {
			const retryNumber uint8 = 0
			currentOffset := (b.offset * maxBlocksToRequest)
			workers[availablePeer] = newSyncerWorkerByNumber(
				availablePeer, uint(currentOffset+1), uint(currentOffset+129),
				bootstrapRequestData, network.Ascending, retryNumber, b.network)
			b.offset++
		}

		workerManager := newWorkerManager(workers)
		blocksData := workerManager.Start()

		sort.Slice(blocksData, func(i, j int) bool {
			return blocksData[i].Header.Number < blocksData[j].Header.Number
		})

		fmt.Printf("====> GOT %d\n", len(blocksData))

		for _, block := range blocksData {
			err := b.processBlockData(*block)
			if err != nil {
				// depending on the error, we might want to save this block for later
				if !errors.Is(err, errFailedToGetParent) && !errors.Is(err, blocktree.ErrParentNotFound) {
					logger.Errorf("block data processing for block with hash %s failed: %s", block.Hash, err)
					continue
				}

				header, bestBlockHeaderErr := b.blockState.BestBlockHeader()
				if bestBlockHeaderErr != nil {
					logger.Errorf("failed to get best block header: %s", bestBlockHeaderErr)
					panic(bestBlockHeaderErr)
				}

				logger.Errorf("block data processing for block with hash %s failed: %s", block.Hash, err)
				logger.Errorf("OFFSET IS AT: %d", b.offset)

				b.offset = uint(header.Number / 128)
				logger.Errorf("NEW OFFSET IS AT: %d", b.offset)
				break
			}
		}
	}
}

// processBlockData processes the BlockData from a BlockResponse and
// returns the index of the last BlockData it handled on success,
// or the index of the block data that errored on failure.
func (b *bootstrapSyncer) processBlockData(blockData types.BlockData) error { //nolint:revive
	logger.Debugf("processing block data with hash %s", blockData.Hash)

	headerInState, err := b.blockState.HasHeader(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has header: %w", err)
	}

	bodyInState, err := b.blockState.HasBlockBody(blockData.Hash)
	if err != nil {
		return fmt.Errorf("checking if block state has body: %w", err)
	}

	// while in bootstrap mode we don't need to broadcast block announcements
	const announceImportedBlock = false
	if headerInState && bodyInState {
		err = b.processBlockDataWithStateHeaderAndBody(blockData, announceImportedBlock)
		if err != nil {
			return fmt.Errorf("processing block data with header and "+
				"body in block state: %w", err)
		}
		return nil
	}

	if blockData.Header != nil {
		if blockData.Body != nil {
			err = b.processBlockDataWithHeaderAndBody(blockData, announceImportedBlock)
			if err != nil {
				return fmt.Errorf("processing block data with header and body: %w", err)
			}
			logger.Debugf("block with hash %s processed", blockData.Hash)
		}

		if blockData.Justification != nil && len(*blockData.Justification) > 0 {
			err = b.handleJustification(blockData.Header, *blockData.Justification)
			if err != nil {
				return fmt.Errorf("handling justification: %w", err)
			}
		}
	}

	err = b.blockState.CompareAndSetBlockData(&blockData)
	if err != nil {
		return fmt.Errorf("comparing and setting block data: %w", err)
	}

	return nil
}

func (b *bootstrapSyncer) processBlockDataWithStateHeaderAndBody(blockData types.BlockData, //nolint:revive
	announceImportedBlock bool) (err error) {
	// TODO: fix this; sometimes when the node shuts down the "best block" isn't stored properly,
	// so when the node restarts it has blocks higher than what it thinks is the best, causing it not to sync
	// if we update the node to only store finalised blocks in the database, this should be fixed and the entire
	// code block can be removed (#1784)
	block, err := b.blockState.GetBlockByHash(blockData.Hash)
	if err != nil {
		return fmt.Errorf("getting block by hash: %w", err)
	}

	err = b.blockState.AddBlockToBlockTree(block)
	if errors.Is(err, blocktree.ErrBlockExists) {
		logger.Debugf(
			"block number %d with hash %s already exists in block tree, skipping it.",
			block.Header.Number, blockData.Hash)
		return nil
	} else if err != nil {
		return fmt.Errorf("adding block to blocktree: %w", err)
	}

	if blockData.Justification != nil && len(*blockData.Justification) > 0 {
		err = b.handleJustification(&block.Header, *blockData.Justification)
		if err != nil {
			return fmt.Errorf("handling justification: %w", err)
		}
	}

	// TODO: this is probably unnecessary, since the state is already in the database
	// however, this case shouldn't be hit often, since it's only hit if the node state
	// is rewinded or if the node shuts down unexpectedly (#1784)
	state, err := b.storageState.TrieState(&block.Header.StateRoot)
	if err != nil {
		return fmt.Errorf("loading trie state: %w", err)
	}

	err = b.blockImportHandler.HandleBlockImport(block, state, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block import: %w", err)
	}

	return nil
}

func (b *bootstrapSyncer) processBlockDataWithHeaderAndBody(blockData types.BlockData, //nolint:revive
	announceImportedBlock bool) (err error) {
	err = b.babeVerifier.VerifyBlock(blockData.Header)
	if err != nil {
		return fmt.Errorf("babe verifying block: %w", err)
	}

	b.handleBody(blockData.Body)

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	err = b.handleBlock(block, announceImportedBlock)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	return nil
}

// handleHeader handles block bodies included in BlockResponses
func (b *bootstrapSyncer) handleBody(body *types.Body) {
	for _, ext := range *body {
		b.transactionState.RemoveExtrinsic(ext)
	}
}

var errFailedToGetParent = errors.New("failed to get parent header")

// handleHeader handles blocks (header+body) included in BlockResponses
func (b *bootstrapSyncer) handleBlock(block *types.Block, announceImportedBlock bool) error {
	parent, err := b.blockState.GetHeader(block.Header.ParentHash)
	if err != nil {
		return fmt.Errorf("%w: %s", errFailedToGetParent, err)
	}

	b.storageState.Lock()
	defer b.storageState.Unlock()

	ts, err := b.storageState.TrieState(&parent.StateRoot)
	if err != nil {
		return err
	}

	root := ts.MustRoot()
	if !bytes.Equal(parent.StateRoot[:], root[:]) {
		panic("parent state root does not match snapshot state root")
	}

	rt, err := b.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	_, err = rt.ExecuteBlock(block)
	if err != nil {
		return fmt.Errorf("failed to execute block %d: %w", block.Header.Number, err)
	}

	if err = b.blockImportHandler.HandleBlockImport(block, ts, announceImportedBlock); err != nil {
		return err
	}

	logger.Debugf("🔗 imported block number %d with hash %s", block.Header.Number, block.Header.Hash())

	blockHash := block.Header.Hash()
	b.telemetry.SendMessage(telemetry.NewBlockImport(
		&blockHash,
		block.Header.Number,
		"NetworkInitialSync"))

	return nil
}

func (b *bootstrapSyncer) handleJustification(header *types.Header, justification []byte) (err error) {
	logger.Debugf("handling justification for block %d...", header.Number)

	headerHash := header.Hash()
	err = b.finalityGadget.VerifyBlockJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("verifying block number %d justification: %w", header.Number, err)
	}

	err = b.blockState.SetJustification(headerHash, justification)
	if err != nil {
		return fmt.Errorf("setting justification for block number %d: %w", header.Number, err)
	}

	logger.Infof("🔨 finalised block number %d with hash %s", header.Number, headerHash)
	return nil
}
