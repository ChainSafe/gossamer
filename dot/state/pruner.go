package state

import (
	"fmt"
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
	StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error
}

// ArchivalNodePruner is a no-op since we don't prune nodes in archive mode.
type ArchivalNodePruner struct{}

// StoreJournalRecord for archive node doesn't do anything.
func (a *ArchivalNodePruner) StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error {
	return nil
}

type deathRecord struct {
	blockHash   common.Hash
	deletedKeys map[common.Hash]int64 // key hash that will be deleted from DB
}

type deathRow []*deathRecord

// FullNodePruner stores state trie diff and allows online state trie pruning
type FullNodePruner struct {
	deathList     []deathRow
	storageDB     chaindb.Database
	journalDB     chaindb.Database
	deathIndex    map[common.Hash]int64
	pendingNumber int64
	retainBlocks  int64
	sync.RWMutex
}

type journalRecord struct {
	// blockHash of the block corresponding to journal record
	blockHash common.Hash
	// Hash of keys that are inserted into state trie of the block
	insertedKeys []common.Hash
	// Hash of keys that are deleted from state trie of the block
	deletedKeys []common.Hash
}

func newJournalRecord(hash common.Hash, insertedKeys, deletedKeys []common.Hash) *journalRecord {
	return &journalRecord{
		blockHash:    hash,
		insertedKeys: insertedKeys,
		deletedKeys:  deletedKeys,
	}
}

// CreatePruner creates a pruner
func CreatePruner(db chaindb.Database, retainBlocks int64) (Pruner, error) {
	p := &FullNodePruner{
		deathList:    make([]deathRow, 0),
		deathIndex:   make(map[common.Hash]int64),
		storageDB:    chaindb.NewTable(db, storagePrefix),
		journalDB:    chaindb.NewTable(db, journalPrefix),
		retainBlocks: retainBlocks,
	}

	blockNum, err := p.getLastPrunedIndex()
	if err != nil {
		return nil, err
	}

	logger.Info("last pruned block", "block num", blockNum)
	blockNum++

	p.pendingNumber = blockNum

	err = p.loadDeathList()
	if err != nil {
		return nil, err
	}

	go p.start()

	return p, nil
}

// StoreJournalRecord stores journal record into DB and add deathRow into deathList
func (p *FullNodePruner) StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error {
	jr := newJournalRecord(blockHash, inserted, deleted)

	key := &journalKey{blockNum, blockHash}
	err := p.storeJournal(key, jr)
	if err != nil {
		return fmt.Errorf("failed to store journal record for %d: %w", blockNum, err)
	}

	logger.Info("journal record stored", "block num", blockNum)
	p.addDeathRow(jr, blockNum)
	return nil
}

func (p *FullNodePruner) addDeathRow(jr *journalRecord, blockNum int64) {
	if blockNum == 0 {
		return
	}

	p.Lock()
	defer p.Unlock()

	if blockNum < p.pendingNumber {
		return
	}

	p.processInsertedKeys(jr.insertedKeys, jr.blockHash)

	// add deleted keys from journal to death index
	for _, k := range jr.deletedKeys {
		p.deathIndex[k] = blockNum
	}

	deletedKeys := make(map[common.Hash]int64)
	for _, data := range jr.deletedKeys {
		deletedKeys[data] = blockNum
	}

	blockIndex := blockNum - p.pendingNumber
	for idx := blockIndex - int64(len(p.deathList)); idx >= 0; idx-- {
		p.deathList = append(p.deathList, deathRow{})
	}

	record := &deathRecord{
		blockHash:   jr.blockHash,
		deletedKeys: deletedKeys,
	}

	// add deathRow to deathList
	p.deathList[blockIndex] = append(p.deathList[blockIndex], record)
}

// Remove re-inserted keys
func (p *FullNodePruner) processInsertedKeys(insKeys []common.Hash, blockHash common.Hash) {
	for _, k := range insKeys {
		if num, ok := p.deathIndex[k]; ok {
			records := p.deathList[num-p.pendingNumber]
			for _, v := range records {
				if v.blockHash == blockHash {
					delete(v.deletedKeys, k)
				}
			}
			delete(p.deathIndex, k)
		}
	}
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

		// pop first element from death list
		row := p.deathList[0]
		blockNum := p.pendingNumber
		if len(row) == 0 {
			logger.Info("death row is empty", "block num", p.pendingNumber)
			continue
		}

		blockHash := row[0].blockHash
		logger.Info("pruning block", "block num", blockNum)

		for _, record := range row {
			err := p.deleteKeys(record.deletedKeys)
			if err != nil {
				logger.Error("failed to prune keys", "block num", blockNum, "error", err)
				continue
			}

			for k := range record.deletedKeys {
				delete(p.deathIndex, k)
			}
		}

		err := p.storeLastPrunedIndex(blockNum)
		if err != nil {
			logger.Error("failed to store last pruned index", "block num", blockNum, "error", err)
			continue
		}

		p.deathList = p.deathList[1:]

		jk := &journalKey{blockNum, blockHash}
		err = p.deleteJournalRecord(jk)
		if err != nil {
			logger.Error("failed to delete journal record", "block num", blockNum, "error", err)
			continue
		}

		p.pendingNumber++
		p.Unlock()
	}
}

type journalKey struct {
	blockNum  int64
	blockHash common.Hash
}

func (p *FullNodePruner) storeJournal(key *journalKey, jr *journalRecord) error {
	encKey, err := scale.Encode(key)
	if err != nil {
		return fmt.Errorf("failed to encode journal key block num %d: %w", key.blockNum, err)
	}

	encRecord, err := scale.Encode(jr)
	if err != nil {
		return fmt.Errorf("failed to encode journal record block num %d: %w", key.blockNum, err)
	}

	err = p.journalDB.Put(encKey, encRecord)
	if err != nil {
		return err
	}

	return nil
}

// loadDeathList loads deathList and deathIndex from journalRecord.
func (p *FullNodePruner) loadDeathList() error {
	itr := p.journalDB.NewIterator()
	defer itr.Release()

	for itr.Next() {
		jk, err := scale.Decode(itr.Key(), new(journalKey))
		if err != nil {
			return fmt.Errorf("failed to decode journal key %w", err)
		}

		key := jk.(*journalKey)
		val := itr.Value()

		jr, err := scale.Decode(val, new(journalRecord))
		if err != nil {
			return fmt.Errorf("failed to decode journal record block num %d : %w", key.blockNum, err)
		}

		p.addDeathRow(jr.(*journalRecord), key.blockNum)
	}
	return nil
}

func (p *FullNodePruner) deleteJournalRecord(key *journalKey) error {
	encKey, err := scale.Encode(key)
	if err != nil {
		return err
	}

	err = p.journalDB.Del(encKey)
	if err != nil {
		return err
	}

	return nil
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

func (p *FullNodePruner) deleteKeys(nodesHash map[common.Hash]int64) error {
	for k := range nodesHash {
		err := p.storageDB.Del(k.ToBytes())
		if err != nil {
			return err
		}
	}

	return nil
}
