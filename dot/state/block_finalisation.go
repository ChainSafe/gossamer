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
	header, err := bs.GetFinalisedHeader(0, 0)
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
		return 0, 0, err
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

// SetFinalisedHash sets the latest finalised block header
func (bs *BlockState) SetFinalisedHash(hash common.Hash, round, setID uint64) error {
	bs.Lock()
	defer bs.Unlock()

	has, _ := bs.HasHeader(hash)
	if !has {
		return fmt.Errorf("cannot finalise unknown block %s", hash)
	}

	// if nothing was previously finalised, set the first slot of the network to the
	// slot number of block 1, which is now being set as final
	if bs.lastFinalised.Equal(bs.genesisHash) && !hash.Equal(bs.genesisHash) {
		err := bs.setFirstSlotOnFinalisation()
		if err != nil {
			return err
		}
	}

	if round > 0 {
		bs.notifyFinalized(hash, round, setID)
	}

	pruned := bs.bt.Prune(hash)
	for _, hash := range pruned {
		header, err := bs.GetHeader(hash)
		if err != nil {
			logger.Debug("failed to get pruned header", "hash", hash, "error", err)
			continue
		}

		err = bs.DeleteBlock(hash)
		if err != nil {
			logger.Debug("failed to delete block", "hash", hash, "error", err)
			continue
		}

		logger.Trace("pruned block", "hash", hash, "number", header.Number)
		go func(header *types.Header) {
			bs.pruneKeyCh <- header
		}(header)
	}

	bs.lastFinalised = hash

	if err := bs.db.Put(finalisedHashKey(round, setID), hash[:]); err != nil {
		return err
	}

	return bs.setHighestRoundAndSetID(round, setID)
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
