package state

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

const (
	journalPrefix = "journal"
	lastPruned    = "last_pruned"
)

type journalRecord struct {
	blockHash *common.Hash
	// Hash of keys that are inserted into state trie of the block
	insertedKeys []*common.Hash
	// Hash of keys that are deleted from state trie of the block
	deletedKeys []*common.Hash
}

type deathRow struct {
	blockHash   *common.Hash
	deletedKeys map[*common.Hash]int64 // keys hash that will be deleted from DB
}

// Pruner stores state trie diff and allows online state trie pruning
type Pruner struct {
	deathList     []*deathRow
	storageDB     chaindb.Database
	journalDB     chaindb.Database
	deathIndex    map[*common.Hash]int64
	pendingNumber int64
	retainBlocks  int64
	sync.RWMutex
}

func newJournalRecord(hash *common.Hash, insertedKeys, deletedKeys []*common.Hash) *journalRecord {
	return &journalRecord{
		blockHash:    hash,
		insertedKeys: insertedKeys,
		deletedKeys:  deletedKeys,
	}
}

// CreatePruner creates a pruner
func CreatePruner(db chaindb.Database, retainBlocks int64) (*Pruner, error) {
	p := &Pruner{
		deathList:    make([]*deathRow, 0),
		deathIndex:   make(map[*common.Hash]int64),
		storageDB:    chaindb.NewTable(db, storagePrefix),
		journalDB:    chaindb.NewTable(db, journalPrefix),
		retainBlocks: retainBlocks,
	}

	blockNum, err := p.getLastPrunedIndex()
	if err != nil {
		if err == chaindb.ErrKeyNotFound {
			blockNum = 0
		} else {
			logger.Error("pruner", "failed to get last pruned key", err)
			return nil, err
		}
	}

	logger.Info("pruner", "last pruned block", blockNum)
	blockNum++

	p.pendingNumber = blockNum

	// load deathList and deathIndex from journalRecord
	for {
		record, err := p.getJournalRecord(blockNum)
		if err != nil {
			if err == chaindb.ErrKeyNotFound {
				break
			}
			return nil, err
		}

		jr, err := scale.Decode(record, new(journalRecord))
		if err != nil {
			return nil, err
		}

		j := jr.(journalRecord)
		err = p.addDeathRow(&j, blockNum)
		if err != nil {
			return nil, err
		}

		blockNum++
	}

	return p, nil
}

// StoreJournalRecord stores journal record into DB and add deathRow into deathList
func (p *Pruner) StoreJournalRecord(deleted, inserted []*common.Hash, blockHash *common.Hash, blockNum *big.Int) error {
	jr := newJournalRecord(blockHash, inserted, deleted)
	encRecord, err := scale.Encode(jr)
	if err != nil {
		return fmt.Errorf("failed to encode journal record %d: %w", blockNum, err)
	}

	err = p.storeJournal(blockNum.Int64(), encRecord)
	if err != nil {
		return fmt.Errorf("failed to store journal record for %d: %w", blockNum, err)
	}

	logger.Info("journal record stored")
	err = p.addDeathRow(jr, blockNum.Int64())
	if err != nil {
		return err
	}

	return nil
}

func (p *Pruner) addDeathRow(jr *journalRecord, blockNum int64) error {
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

	deletedKeys := make(map[*common.Hash]int64)
	for _, data := range jr.deletedKeys {
		deletedKeys[data] = blockNum
	}

	dr := &deathRow{
		blockHash:   jr.blockHash,
		deletedKeys: deletedKeys,
	}

	// add deathRow to deathList
	p.deathList = append(p.deathList, dr)
	return nil
}

// PruneOne starts online pruning process
func (p *Pruner) PruneOne() {
	logger.Info("pruning started")

	for {
		if int64(len(p.deathList)) <= p.retainBlocks {
			time.Sleep(2 * time.Second)
			continue
		}

		logger.Info("pruner", "pruning block ", p.pendingNumber)
		p.Lock()

		// pop first element from death list
		dr := p.deathList[0]
		err := p.deleteKeys(dr.deletedKeys)
		if err != nil {
			logger.Error("pruner", "failed to delete keys for block", p.pendingNumber)
			continue
		}

		for k := range dr.deletedKeys {
			delete(p.deathIndex, k)
		}

		err = p.storeLastPrunedIndex(p.pendingNumber)
		if err != nil {
			logger.Error("pruner", "failed to store last pruned index")
		}

		p.deathList = p.deathList[1:]
		p.pendingNumber++
		p.Unlock()
	}
}

func (p *Pruner) storeJournal(num int64, record []byte) error {
	encNum, err := scale.Encode(num)
	if err != nil {
		return err
	}

	err = p.journalDB.Put(encNum, record)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pruner) getJournalRecord(num int64) ([]byte, error) {
	encNum, err := scale.Encode(num)
	if err != nil {
		return nil, err
	}

	val, err := p.journalDB.Get(encNum)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (p *Pruner) storeLastPrunedIndex(blockNum int64) error {
	encNum, err := scale.Encode(blockNum)
	if err != nil {
		return err
	}

	err = p.journalDB.Put([]byte(lastPruned), encNum)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pruner) getLastPrunedIndex() (int64, error) {
	val, err := p.journalDB.Get([]byte(lastPruned))
	if err != nil {
		return 0, err
	}

	blockNum, err := scale.Decode(val, int64(0))
	if err != nil {
		return 0, err
	}

	return blockNum.(int64), err
}

func (p *Pruner) deleteKeys(nodesHash map[*common.Hash]int64) error {
	for k := range nodesHash {
		err := p.storageDB.Del(k.ToBytes())
		if err != nil {
			return err
		}
	}

	return nil
}
