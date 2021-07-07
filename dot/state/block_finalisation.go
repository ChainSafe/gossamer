package state

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// finalizedHashKey = FinalizedBlockHashKey + round + setID (LE encoded)
func finalizedHashKey(round, setID uint64) []byte {
	return append(common.FinalizedBlockHashKey, roundSetIDKey(round, setID)...)
}

// func roundSetIDKey(round, setID uint64) []byte {
// 	buf := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(buf, round)
// 	buf2 := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(buf2, setID)
// 	return append(buf, buf2...)
// }

// // HasPrevotes returns if the db contains prevotes for the given round and set ID
// func (bs *BlockState) HasPrevotes(round, setID uint64) (bool, error) {
// 	return bs.db.Has(prevotesKey(round, setID))
// }

// // SetPrevotes sets the prevotes for a specific round and set ID in the database
// func (bs *BlockState) SetPrevotes(round, setID uint64, data []byte) error {
// 	return bs.db.Put(prevotesKey(round, setID), data)
// }

// // GetPrevotes retrieves the prevotes for a specific round and set ID from the database
// func (bs *BlockState) GetPrevotes(round, setID uint64) ([]byte, error) {
// 	return bs.db.Get(prevotesKey(round, setID))
// }

// // HasPrecommits returns if the db contains precommits for the given round and set ID
// func (bs *BlockState) HasPrecommits(round, setID uint64) (bool, error) {
// 	return bs.db.Has(precommitsKey(round, setID))
// }

// // SetPrecommits sets the precommits for a specific round and set ID in the database
// func (bs *BlockState) SetPrecommits(round, setID uint64, data []byte) error {
// 	return bs.db.Put(precommitsKey(round, setID), data)
// }

// // GetPrecommits retrieves the precommits for a specific round and set ID from the database
// func (bs *BlockState) GetPrecommits(round, setID uint64) ([]byte, error) {
// 	return bs.db.Get(precommitsKey(round, setID))
// }

// HasFinalizedBlock returns true if there is a finalised block for a given round and setID, false otherwise
func (bs *BlockState) HasFinalizedBlock(round, setID uint64) (bool, error) {
	return bs.db.Has(finalizedHashKey(round, setID))
}

// NumberIsFinalised checks if a block number is finalised or not
func (bs *BlockState) NumberIsFinalised(num *big.Int) (bool, error) {
	header, err := bs.GetFinalizedHeader(0, 0)
	if err != nil {
		return false, err
	}

	return num.Cmp(header.Number) <= 0, nil
}

// GetFinalizedHeader returns the finalised block header by round and setID
func (bs *BlockState) GetFinalizedHeader(round, setID uint64) (*types.Header, error) {
	h, err := bs.GetFinalizedHash(round, setID)
	if err != nil {
		return nil, err
	}

	header, err := bs.GetHeader(h)
	if err != nil {
		return nil, err
	}

	return header, nil
}

// GetFinalizedHash gets the finalised block header by round and setID
func (bs *BlockState) GetFinalizedHash(round, setID uint64) (common.Hash, error) {
	h, err := bs.db.Get(finalizedHashKey(round, setID))
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHash(h), nil
}

// SetFinalizedHash sets the latest finalised block header
// Note that using round=0 and setID=0 would refer to the latest finalized hash
func (bs *BlockState) SetFinalizedHash(hash common.Hash, round, setID uint64) error {
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
		go bs.notifyFinalized(hash, round, setID)
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
	return bs.db.Put(finalizedHashKey(round, setID), hash[:])
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
