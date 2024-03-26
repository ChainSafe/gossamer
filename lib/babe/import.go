package babe

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
)

type BlockOrigin byte

const (
	NetworkInitialSync BlockOrigin = iota
	NetworkBroadcast
	Own
)

type Backend interface {
	StoreTrie(*rtstorage.TrieState)
	AddBlock(*types.Block)
	HandleDigests(*types.Header)
	ApplyForcedChanges(*types.Header)
	HandleRuntimeChanges()
}

type BlockImporter struct {
	transactionState TransactionState
	epochState       EpochState
	slotState        SlotState
	blockState       BlockState
}

type BlockImportParams struct {
	Block    *types.Block
	Origin   BlockOrigin
	Announce bool
}

// while importing check if block is 1 (obviously if the lastest one is genesis, otherwise just ignore the block)
// if block is 1 then:
//
//	 #  store its slot in a in-memory map [hash] -> slot number until finalization happens,
//		then the very first slot can be retrieved trivialy.
//		it should contains the next epoch descriptor, then save it.
//		with the start slot from the epoch 0 calculates the end of epoch 0 (this is needed to check when the next epoch starts)
//
// if block is not 1 then:
//
//	 #  check if the block starts a new epoch, if so check if it contains the next epoch descriptor, if not reject it
//		if it starts a new epoch then calculates the end of this epoch
//		if the epoch calculation overpass more then 1 epoch, then use the next already setup epoch descriptor

func (b *BlockImporter) Import(block *types.Block, state *rtstorage.TrieState) error {
	parentHash := block.Header.ParentHash
	parentBlock, err := b.blockState.GetBlockByHash(parentHash)
	if err != nil {
		return fmt.Errorf("getting parent block by hash: %w", err)
	}

	parentBlockSlot, err := parentBlock.Header.SlotNumber()
	if err != nil {
		return fmt.Errorf("getting parent slot number: %w", err)
	}

	currentBlockSlot, err := block.Header.SlotNumber()
	if err != nil {
		return fmt.Errorf("while getting slot number: %w", err)
	}

	// ensure that the slot increase
	if currentBlockSlot <= parentBlockSlot {
		return fmt.Errorf("slot must increase, current block slot %d, parent block slot %d",
			currentBlockSlot, parentBlockSlot)
	}

	//b.epochState.GetEpochDescriptorFor(block.Header)

	return nil
}

func (b *BlockImporter) verify() {

}
