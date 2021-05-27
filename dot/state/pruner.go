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
	lastPrunedKey = "last_pruned"
)

// Pruner is implemented by FullNodePruner and ArchivalNodePruner.
type Pruner interface {
	StoreJournalRecord(deleted, inserted []*common.Hash, blockHash *common.Hash, blockNum *big.Int) error
}

// ArchivalNodePruner is a no-op since we don't prune nodes in archive mode.
type ArchivalNodePruner struct{}

// StoreJournalRecord for archive node doesn't do anything.
func (a *ArchivalNodePruner) StoreJournalRecord(deleted, inserted []*common.Hash, blockHash *common.Hash, blockNum *big.Int) error {
	return nil
}

// FullNodePruner stores state trie diff and allows online state trie pruning
type FullNodePruner struct {
	deathList     []*deathRow
	storageDB     chaindb.Database
	journalDB     chaindb.Database
	deathIndex    map[*common.Hash]int64
	pendingNumber int64
	retainBlocks  int64
	sync.RWMutex
}

type journalRecord struct {
	// blockHash of the block corresponding to journal record
	blockHash *common.Hash
	// Hash of keys that are inserted into state trie of the block
	insertedKeys []*common.Hash
	// Hash of keys that are deleted from state trie of the block
	deletedKeys []*common.Hash
}

func newJournalRecord(hash *common.Hash, insertedKeys, deletedKeys []*common.Hash) *journalRecord {
	return &journalRecord{
		blockHash:    hash,
		insertedKeys: insertedKeys,
		deletedKeys:  deletedKeys,
	}
}

type deathRow struct {
	blockHash   *common.Hash
	deletedKeys map[*common.Hash]int64 // keys hash that will be deleted from DB
}

// CreatePruner creates a pruner
func CreatePruner(db chaindb.Database, retainBlocks int64) (Pruner, error) {
	p := &FullNodePruner{
		deathList:    make([]*deathRow, 0),
		deathIndex:   make(map[*common.Hash]int64),
		storageDB:    chaindb.NewTable(db, storagePrefix),
		journalDB:    chaindb.NewTable(db, journalPrefix),
		retainBlocks: retainBlocks,
	}

	blockNum, err := p.getLastPrunedIndex()
	if err != nil {
		return nil, err
	}

	logger.Info("pruner", "last pruned block", blockNum)
	blockNum++

	p.pendingNumber = blockNum

	// load deathList and deathIndex from journalRecord
	for {
		record, err := p.getJournalRecord(blockNum)
		if err == chaindb.ErrKeyNotFound {
			break
		}

		if err != nil {
			return nil, err
		}

		err = p.addDeathRow(record, blockNum)
		if err != nil {
			return nil, err
		}

		blockNum++
	}

	go p.start()

	return p, nil
}

// StoreJournalRecord stores journal record into DB and add deathRow into deathList
func (p *FullNodePruner) StoreJournalRecord(deleted, inserted []*common.Hash, blockHash *common.Hash, blockNum *big.Int) error {
	jr := newJournalRecord(blockHash, inserted, deleted)

	err := p.storeJournal(blockNum.Int64(), jr)
	if err != nil {
		return fmt.Errorf("failed to store journal record for %d: %w", blockNum, err)
	}

	logger.Info("journal record stored", "block", blockNum.Int64())
	err = p.addDeathRow(jr, blockNum.Int64())
	if err != nil {
		return err
	}

	return nil
}

func (p *FullNodePruner) addDeathRow(jr *journalRecord, blockNum int64) error {
	p.Lock()
	defer p.Unlock()

	// remove re-inserted keys
	for _, k := range jr.insertedKeys {
		if num, ok := p.deathIndex[k]; ok {
			delete(p.deathList[num-p.pendingNumber].deletedKeys, k)
			delete(p.deathIndex, k)
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

func (p *FullNodePruner) start() {
	logger.Info("pruning started")

	for {
		p.Lock()
		if int64(len(p.deathList)) <= p.retainBlocks {
			p.Unlock()
			time.Sleep(2 * time.Second)
			continue
		}

		logger.Info("pruner", "pruning block ", p.pendingNumber)

		// pop first element from death list
		row := p.deathList[0]
		err := p.deleteKeys(row.deletedKeys)
		if err != nil {
			logger.Error("pruner", "failed to delete keys for block", p.pendingNumber)
			continue
		}

		for k := range row.deletedKeys {
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

func (p *FullNodePruner) storeJournal(blockNum int64, jr *journalRecord) error {
	encRecord, err := scale.Encode(jr)
	if err != nil {
		return fmt.Errorf("failed to encode journal record %d: %w", blockNum, err)
	}

	encNum, err := scale.Encode(blockNum)
	if err != nil {
		return err
	}

	err = p.journalDB.Put(encNum, encRecord)
	if err != nil {
		return err
	}

	return nil
}

func (p *FullNodePruner) getJournalRecord(num int64) (*journalRecord, error) {
	encNum, err := scale.Encode(num)
	if err != nil {
		return nil, err
	}

	val, err := p.journalDB.Get(encNum)
	if err != nil {
		return nil, err
	}

	decJR, err := scale.Decode(val, new(journalRecord))
	if err != nil {
		return nil, err
	}

	return decJR.(*journalRecord), nil
}

func (p *FullNodePruner) storeLastPrunedIndex(blockNum int64) error {
	encNum, err := scale.Encode(blockNum)
	if err != nil {
		return err
	}

	err = p.journalDB.Put([]byte(lastPrunedKey), encNum)
	if err != nil {
		return err
	}

	return nil
}

func (p *FullNodePruner) getLastPrunedIndex() (int64, error) {
	val, err := p.journalDB.Get([]byte(lastPrunedKey))
	if err == chaindb.ErrKeyNotFound {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	blockNum, err := scale.Decode(val, int64(0))
	if err != nil {
		return 0, err
	}

	return blockNum.(int64), err
}

func (p *FullNodePruner) deleteKeys(nodesHash map[*common.Hash]int64) error {
	for k := range nodesHash {
		err := p.storageDB.Del(k.ToBytes())
		if err != nil {
			return err
		}
	}

	return nil
}
