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
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// finalisedHashKey = FinalizedBlockHashKey + round + setID (LE encoded)
func finalisedHashKey(round, setID uint64) []byte {
	return append(common.FinalizedBlockHashKey, roundSetIDKey(round, setID)...)
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

// SetFinalisedHash sets the latest finalised block header
// Note that using round=0 and setID=0 would refer to the latest finalised hash
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
	for _, rem := range pruned {
		header, err := bs.GetHeader(rem)
		if err != nil {
			return err
		}

		err = bs.DeleteBlock(rem)
		if err != nil {
			return err
		}

		logger.Trace("pruned block", "hash", rem, "number", header.Number)
		bs.pruneKeyCh <- header
	}

	bs.lastFinalised = hash
	return bs.db.Put(finalisedHashKey(round, setID), hash[:])
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
