package sync

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
)

type JournalRecord struct {
	blockHash *common.Hash
	// Hash of keys that are inserted into state trie of the block
	insertedKeys []*common.Hash
	// Hash of keys that are deleted from state trie of the block
	deletedKeys []*common.Hash

	lock sync.RWMutex
}

type DeathRow struct {
	blockHash *common.Hash
	deletedKeys map[*common.Hash]uint64 // keys hash that will be deleted from DB
	lock        sync.RWMutex
}

type pruner struct {
	deathList      []*DeathRow
	deathIndex     map[*common.Hash]uint64
	pendingNumber  uint64
	sync.Mutex
}

func createJournalRecord(hash *common.Hash, insertedKeys, deletedKeys []*common.Hash) *JournalRecord {
	return &JournalRecord{
		blockHash:    hash,
		insertedKeys: insertedKeys,
		deletedKeys:  deletedKeys,
	}
}

func newPruner(s StorageState) (*pruner, error) {
	blockNum, err := s.GetLastPrunedIndex()
	if err != nil {
		if err == chaindb.ErrKeyNotFound {
			blockNum = 0
		}
		return nil, err
	}

	blockNum += 1

	p := &pruner{
		deathList:     make([]*DeathRow, 0),
		deathIndex:    make(map[*common.Hash]uint64, 0),
		pendingNumber: blockNum,
	}

	// load deathList and deathIndex from JournalRecord
	for {
		record, err := s.GetJournalRecord(blockNum)
		if err != nil {
			if err == chaindb.ErrKeyNotFound {
				break
			}
			return nil, err
		}

		jr := &JournalRecord{}
		_, err = scale.Decode(record, jr)
		if err != nil {
			return nil, err
		}


		err = p.addDeathRow(jr, blockNum)
		if err != nil {
			return nil, err
		}

		blockNum += 1
	}

	return p, nil
}

func (p *pruner) storeJournalRecord(ts *storage.TrieState, s StorageState, blockHash *common.Hash, blockNum *big.Int) error {
	insKeys, err := ts.GetInsertedNodeHashes()
	if err != nil {
		return fmt.Errorf("failed to get inserted keys for %d: %w", blockNum, err)
	}

	delKeys := ts.GetDeletedNodeHashes()

	jr := createJournalRecord(blockHash, insKeys, delKeys)

	encRecord, err := scale.Encode(jr)
	if err != nil {
		return fmt.Errorf("failed to encode journal record %d: %w", blockNum, err)
	}

	err = s.StoreJournal(blockNum.Uint64(), encRecord)
	if err != nil {
		return fmt.Errorf("failed to store journal record for %d: %w", blockNum, err)
	}

	err = p.addDeathRow(jr, blockNum.Uint64())
	if err != nil {
		return err
	}

	return nil
}

func (p *pruner) addDeathRow(jr *JournalRecord, blockNum uint64) error {
	p.Lock()
	defer p.Unlock()

	// remove re-inserted keys
	for _, k := range jr.insertedKeys {
		if num, ok := p.deathIndex[k]; ok {
			delete(p.deathIndex, k)
			delete(p.deathList[num-p.pendingNumber].deletedKeys, k)
		}
	}

	// add deleted keys from journal to death index
	for _, k := range jr.deletedKeys {
		p.deathIndex[k] = blockNum
	}

	deletedKeys := make(map[*common.Hash]uint64)
	for _, data := range jr.deletedKeys {
		deletedKeys[data] = blockNum
	}

	dr := &DeathRow{
		blockHash:   jr.blockHash,
		deletedKeys: deletedKeys,
	}

	// add DeathRow to deathList
	p.deathList = append(p.deathList, dr)
	return nil
}

func (p *pruner) pruneOne(s StorageState) {
	p.Lock()
	defer p.Unlock()

	for {
		if len(p.deathList) < 1 {
			logger.Error("%s", "trying to prune when there's nothing to prune")
		}

		// pop first element from death list
		dr := p.deathList[0]
		err := s.DeleteKeys(dr.deletedKeys)
		if err != nil {
			logger.Error("pruner failed to delete keys")
			continue
		}

		for k := range dr.deletedKeys {
			delete(p.deathIndex, k)
		}

		err = s.StoreLastPrunedIndex(p.pendingNumber) //TODO: change lastPrunedIndex to lastPrunedIndex + 1
		if err != nil {
			logger.Error("pruner failed to store last pruned index")
			continue
		}

		p.deathList = p.deathList[1:]
		//p.pendingPruning += 1
		p.pendingNumber += 1
	}
}
