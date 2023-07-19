// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var errSetIDLowerThanHighest = errors.New("set id lower than highest")
var highestRoundAndSetIDKey = []byte("hrs")

// finalisedHashKey = FinalizedBlockHashKey + round + setID (LE encoded)
func finalisedHashKey(round, setID uint64) []byte {
	return append(common.FinalizedBlockHashKey, roundAndSetIDToBytes(round, setID)...)
}

// HasFinalisedBlock returns true if there is a finalised block for a given round and setID, false otherwise
func (bs *BlockState) HasFinalisedBlock(round, setID uint64) (bool, error) {
	return bs.db.Has(finalisedHashKey(round, setID))
}

// NumberIsFinalised checks if a block number is finalised or not
func (bs *BlockState) NumberIsFinalised(num uint) (bool, error) {
	header, err := bs.GetHighestFinalisedHeader()
	if err != nil {
		return false, err
	}

	return num <= header.Number, nil
}

// GetFinalisedHeader returns the finalised block header by round and setID
func (bs *BlockState) GetFinalisedHeader(round, setID uint64) (*types.Header, error) {
	bs.Lock()
	defer bs.Unlock()

	h, err := bs.GetFinalisedHash(round, setID)
	if err != nil {
		return nil, err
	}

	header, err := bs.GetHeader(h)
	if err != nil {
		return nil, err
	}

	return header, nil
}

// GetFinalisedHash gets the finalised block header by round and setID
func (bs *BlockState) GetFinalisedHash(round, setID uint64) (common.Hash, error) {
	h, err := bs.db.Get(finalisedHashKey(round, setID))
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHash(h), nil
}

func (bs *BlockState) setHighestRoundAndSetID(round, setID uint64) error {
	_, highestSetID, err := bs.GetHighestRoundAndSetID()
	if err != nil {
		return err
	}

	if setID < highestSetID {
		return fmt.Errorf("%w: %d should be greater or equal %d", errSetIDLowerThanHighest, setID, highestSetID)
	}

	return bs.db.Put(highestRoundAndSetIDKey, roundAndSetIDToBytes(round, setID))
}

// GetHighestRoundAndSetID gets the highest round and setID that have been finalised
func (bs *BlockState) GetHighestRoundAndSetID() (uint64, uint64, error) {
	b, err := bs.db.Get(highestRoundAndSetIDKey)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get highest round and setID: %w", err)
	}

	round := binary.LittleEndian.Uint64(b[:8])
	setID := binary.LittleEndian.Uint64(b[8:16])
	return round, setID, nil
}

// GetHighestFinalisedHash returns the highest finalised block hash
func (bs *BlockState) GetHighestFinalisedHash() (common.Hash, error) {
	round, setID, err := bs.GetHighestRoundAndSetID()
	if err != nil {
		return common.Hash{}, err
	}

	return bs.GetFinalisedHash(round, setID)
}

// GetHighestFinalisedHeader returns the highest finalised block header
func (bs *BlockState) GetHighestFinalisedHeader() (*types.Header, error) {
	h, err := bs.GetHighestFinalisedHash()
	if err != nil {
		return nil, err
	}

	header, err := bs.GetHeader(h)
	if err != nil {
		return nil, err
	}

	return header, nil
}

// SetFinalisedHash sets the latest finalised block hash
func (bs *BlockState) SetFinalisedHash(hash common.Hash, round, setID uint64) error {
	bs.Lock()
	defer bs.Unlock()

	has, err := bs.HasHeader(hash)
	if err != nil {
		return fmt.Errorf("could not check header for hash %s: %w", hash, err)
	}
	if !has {
		return fmt.Errorf("cannot finalise unknown block %s", hash)
	}

	if err := bs.handleFinalisedBlock(hash); err != nil {
		return fmt.Errorf("failed to set finalised subchain in db on finalisation: %w", err)
	}

	if err := bs.db.Put(finalisedHashKey(round, setID), hash[:]); err != nil {
		return fmt.Errorf("failed to set finalised hash key: %w", err)
	}

	if err := bs.setHighestRoundAndSetID(round, setID); err != nil {
		return fmt.Errorf("failed to set highest round and set ID: %w", err)
	}

	if round > 0 {
		bs.notifyFinalized(hash, round, setID)
	}

	pruned := bs.bt.Prune(hash)
	for _, prunedHash := range pruned {
		blockHeader := bs.unfinalisedBlocks.delete(prunedHash)
		if blockHeader == nil {
			continue
		}

		if prunedHash != hash {
			bs.tries.delete(blockHeader.StateRoot)
		}

		logger.Tracef("pruned block number %d with hash %s", blockHeader.Number, prunedHash)
	}

	// if nothing was previously finalised, set the first slot of the network to the
	// slot number of block 1, which is now being set as final
	if bs.lastFinalised == bs.genesisHash && hash != bs.genesisHash {
		if err := bs.setFirstSlotOnFinalisation(); err != nil {
			return fmt.Errorf("failed to set first slot on finalisation: %w", err)
		}
	}

	header, err := bs.GetHeader(hash)
	if err != nil {
		return fmt.Errorf("failed to get finalised header, hash: %s, error: %s", hash, err)
	}

	bs.telemetry.SendMessage(
		telemetry.NewNotifyFinalized(
			header.Hash(),
			fmt.Sprint(header.Number),
		),
	)

	if bs.lastFinalised != hash {
		defer func(lastFinalised common.Hash) {
			err := bs.deleteFromTries(lastFinalised)
			if err != nil {
				logger.Debugf("%v", err)
			}
		}(bs.lastFinalised)
	}

	bs.lastFinalised = hash
	return nil
}

func (bs *BlockState) deleteFromTries(lastFinalised common.Hash) error {
	lastFinalisedHeader, err := bs.GetHeader(lastFinalised)
	if err != nil {
		return fmt.Errorf("unable to retrieve header for last finalised block, hash: %s, err: %s", bs.lastFinalised, err)
	}
	stateRootTrie := bs.tries.get(lastFinalisedHeader.StateRoot)
	if stateRootTrie != nil {
		bs.tries.delete(lastFinalisedHeader.StateRoot)
	} else {
		return fmt.Errorf("unable to find trie with stateroot hash: %s", lastFinalisedHeader.StateRoot)
	}
	return nil
}

func (bs *BlockState) handleFinalisedBlock(curr common.Hash) error {
	if curr == bs.lastFinalised {
		return nil
	}

	subchain, err := bs.RangeInMemory(bs.lastFinalised, curr)
	if err != nil {
		return err
	}

	batch := bs.db.NewBatch()

	// root of subchain is previously finalised block, which has already been stored in the db
	for _, hash := range subchain[1:] {
		if hash == bs.genesisHash {
			continue
		}

		block := bs.unfinalisedBlocks.getBlock(hash)
		if block == nil {
			return fmt.Errorf("failed to find block in unfinalised block map, block=%s", hash)
		}

		if err = bs.SetHeader(&block.Header); err != nil {
			return err
		}

		if err = bs.SetBlockBody(hash, &block.Body); err != nil {
			return err
		}

		arrivalTime, err := bs.bt.GetArrivalTime(hash)
		if err != nil {
			return err
		}

		if err = bs.setArrivalTime(hash, arrivalTime); err != nil {
			return err
		}

		if err = batch.Put(headerHashKey(uint64(block.Header.Number)), hash.ToBytes()); err != nil {
			return err
		}

		// delete from the unfinalisedBlockMap and delete reference to in-memory trie
		blockHeader := bs.unfinalisedBlocks.delete(hash)
		if blockHeader == nil {
			continue
		}

		if hash != curr {
			bs.tries.delete(blockHeader.StateRoot)
		}

		logger.Tracef("cleaned out finalised block from memory; block number %d with hash %s", blockHeader.Number, hash)
	}
	return batch.Flush()
}

func (bs *BlockState) setFirstSlotOnFinalisation() error {
	header, err := bs.GetHeaderByNumber(1)
	if err != nil {
		return err
	}

	slot, err := types.GetSlotFromHeader(header)
	if err != nil {
		return err
	}

	return bs.baseState.storeFirstSlot(slot)
}
