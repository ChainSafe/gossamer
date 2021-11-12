// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

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
func (bs *BlockState) NumberIsFinalised(num *big.Int) (bool, error) {
	header, err := bs.GetHighestFinalisedHeader()
	if err != nil {
		return false, err
	}

	return num.Cmp(header.Number) <= 0, nil
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
	currRound, currSetID, err := bs.GetHighestRoundAndSetID()
	if err != nil {
		return err
	}

	// higher setID takes precedence over round
	if setID < currSetID || setID == currSetID && round <= currRound {
		return nil
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

	has, _ := bs.HasHeader(hash)
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

	if err := bs.handleFinalisedBlock(hash); err != nil {
		return fmt.Errorf("failed to set number->hash mapping on finalisation: %w", err)
	}

	if round > 0 {
		bs.notifyFinalized(hash, round, setID)
	}

	pruned := bs.bt.Prune(hash)
	for _, hash := range pruned {
		block, has := bs.getAndDeleteUnfinalisedBlock(hash)
		if !has {
			continue
		}

		logger.Tracef("pruned block number %s with hash %s", block.Header.Number, hash)

		go func(header *types.Header) {
			bs.pruneKeyCh <- header
		}(&block.Header)
	}

	// if nothing was previously finalised, set the first slot of the network to the
	// slot number of block 1, which is now being set as final
	if bs.lastFinalised.Equal(bs.genesisHash) && !hash.Equal(bs.genesisHash) {
		if err := bs.setFirstSlotOnFinalisation(); err != nil {
			return fmt.Errorf("failed to set first slot on finalisation: %w", err)
		}
	}

	header, err := bs.GetHeader(hash)
	if err != nil {
		return fmt.Errorf("failed to get finalised header, hash: %s, error: %s", hash, err)
	}

	err = telemetry.GetInstance().SendMessage(
		telemetry.NewNotifyFinalizedTM(
			header.Hash(),
			header.Number.String(),
		),
	)
	if err != nil {
		logger.Debugf("could not send 'notify.finalized' telemetry message, error: %s", err)
	}

	bs.lastFinalised = hash
	return nil
}

func (bs *BlockState) handleFinalisedBlock(curr common.Hash) error {
	if curr.Equal(bs.lastFinalised) {
		return nil
	}

	prev, err := bs.GetHighestFinalisedHash()
	if err != nil {
		return fmt.Errorf("failed to get highest finalised hash: %w", err)
	}

	if prev.Equal(curr) {
		return nil
	}

	subchain, err := bs.SubChain(prev, curr)
	if err != nil {
		return err
	}

	batch := bs.db.NewBatch()

	// root of subchain is previously finalised block, which has already been stored in the db
	for _, hash := range subchain[1:] {
		if hash.Equal(bs.genesisHash) {
			continue
		}

		block, has := bs.getUnfinalisedBlock(hash)
		if !has {
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

		if err = batch.Put(headerHashKey(block.Header.Number.Uint64()), hash.ToBytes()); err != nil {
			return err
		}

		// the block will be deleted from the unfinalisedBlockMap in the pruning loop
		// in `SetFinalisedHash()`, which calls this function
	}

	return batch.Flush()
}

func (bs *BlockState) setFirstSlotOnFinalisation() error {
	header, err := bs.GetHeaderByNumber(big.NewInt(1))
	if err != nil {
		return err
	}

	slot, err := types.GetSlotFromHeader(header)
	if err != nil {
		return err
	}

	return bs.baseState.storeFirstSlot(slot)
}
