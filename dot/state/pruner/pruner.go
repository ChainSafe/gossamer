package pruner

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const (
	journalPrefix = "journal"
	lastPrunedKey = "last_pruned"
	pruneInterval = time.Second
)

// nolint
const (
	Full    = Mode("full")
	Archive = Mode("archive")
)

// Mode online pruning mode of historical state tries
type Mode string

// IsValid checks whether the pruning mode is valid
func (p Mode) IsValid() bool {
	switch p {
	case Full:
		return true
	case Archive:
		return true
	default:
		return false
	}
}

// Config holds state trie pruning mode and retained blocks
type Config struct {
	Mode           Mode
	RetainedBlocks int64
}

// Pruner is implemented by FullNode and ArchiveNode.
type Pruner interface {
	StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error
}

// ArchiveNode is a no-op since we don't prune nodes in archive mode.
type ArchiveNode struct{}

// StoreJournalRecord for archive node doesn't do anything.
func (a *ArchiveNode) StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error {
	return nil
}

type deathRecord struct {
	blockHash   common.Hash
	deletedKeys map[common.Hash]int64 // Mapping from deleted key hash to block number.
}

type deathRow []*deathRecord

// FullNode stores state trie diff and allows online state trie pruning
type FullNode struct {
	logger        log.LeveledLogger
	deathList     []deathRow
	storageDB     chaindb.Database
	journalDB     chaindb.Database
	deathIndex    map[common.Hash]int64 // Mapping from deleted key hash to block number.
	pendingNumber int64                 // block number to be pruned. Initial value is set to 1 and is incremented after every block pruning.
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

type journalKey struct {
	blockNum  int64
	blockHash common.Hash
}

func newJournalRecord(hash common.Hash, insertedKeys, deletedKeys []common.Hash) *journalRecord {
	return &journalRecord{
		blockHash:    hash,
		insertedKeys: insertedKeys,
		deletedKeys:  deletedKeys,
	}
}

// NewFullNode creates a Pruner for full node.
func NewFullNode(db, storageDB chaindb.Database, retainBlocks int64, l log.LeveledLogger) (Pruner, error) {
	p := &FullNode{
		deathList:    make([]deathRow, 0),
		deathIndex:   make(map[common.Hash]int64),
		storageDB:    storageDB,
		journalDB:    chaindb.NewTable(db, journalPrefix),
		retainBlocks: retainBlocks,
		logger:       l,
	}

	blockNum, err := p.getLastPrunedIndex()
	if err != nil {
		return nil, err
	}

	p.logger.Debugf("last pruned block is %d", blockNum)
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
func (p *FullNode) StoreJournalRecord(deleted, inserted []common.Hash, blockHash common.Hash, blockNum int64) error {
	jr := newJournalRecord(blockHash, inserted, deleted)

	key := &journalKey{blockNum, blockHash}
	err := p.storeJournal(key, jr)
	if err != nil {
		return fmt.Errorf("failed to store journal record for %d: %w", blockNum, err)
	}

	p.logger.Debugf("journal record stored for block number %d", blockNum)
	p.addDeathRow(jr, blockNum)
	return nil
}

func (p *FullNode) addDeathRow(jr *journalRecord, blockNum int64) {
	if blockNum == 0 {
		return
	}

	p.Lock()
	defer p.Unlock()

	// The block is already pruned.
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
func (p *FullNode) processInsertedKeys(insKeys []common.Hash, blockHash common.Hash) {
	for _, k := range insKeys {
		num, ok := p.deathIndex[k]
		if !ok {
			continue
		}
		records := p.deathList[num-p.pendingNumber]
		for _, v := range records {
			if v.blockHash == blockHash {
				delete(v.deletedKeys, k)
			}
		}
		delete(p.deathIndex, k)
	}
}

func (p *FullNode) start() {
	p.logger.Debug("pruning started")

	var canPrune bool
	checkPruning := func() {
		p.Lock()
		defer p.Unlock()
		if int64(len(p.deathList)) <= p.retainBlocks {
			canPrune = false
			return
		}
		canPrune = true

		// pop first element from death list
		row := p.deathList[0]
		blockNum := p.pendingNumber

		p.logger.Debugf("pruning block number %d", blockNum)

		sdbBatch := p.storageDB.NewBatch()
		for _, record := range row {
			err := p.deleteKeys(sdbBatch, record.deletedKeys)
			if err != nil {
				p.logger.Warnf("failed to prune keys for block number %d: %s", blockNum, err)
				sdbBatch.Reset()
				return
			}

			for k := range record.deletedKeys {
				delete(p.deathIndex, k)
			}
		}

		if err := sdbBatch.Flush(); err != nil {
			p.logger.Warnf("failed to prune keys for block number %d: %s", blockNum, err)
			return
		}

		err := p.storeLastPrunedIndex(blockNum)
		if err != nil {
			p.logger.Warnf("failed to store last pruned index for block number %d: %s", blockNum, err)
			return
		}

		p.deathList = p.deathList[1:]
		p.pendingNumber++

		jdbBatch := p.journalDB.NewBatch()
		for _, record := range row {
			jk := &journalKey{blockNum, record.blockHash}
			err = p.deleteJournalRecord(jdbBatch, jk)
			if err != nil {
				p.logger.Warnf("failed to delete journal record for block number %d: %s", blockNum, err)
				jdbBatch.Reset()
				return
			}
		}

		if err = jdbBatch.Flush(); err != nil {
			p.logger.Warnf("failed to flush delete journal record for block number %d: %s", blockNum, err)
			return
		}
		p.logger.Debugf("pruned block number %d", blockNum)
	}

	for {
		checkPruning()
		// Don't sleep if we have data to prune.
		if !canPrune {
			time.Sleep(pruneInterval)
		}
	}
}

func (p *FullNode) storeJournal(key *journalKey, jr *journalRecord) error {
	encKey, err := scale.Marshal(*key)
	if err != nil {
		return fmt.Errorf("failed to encode journal key block num %d: %w", key.blockNum, err)
	}

	encRecord, err := scale.Marshal(*jr)
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
func (p *FullNode) loadDeathList() error {
	itr := p.journalDB.NewIterator()
	defer itr.Release()

	for itr.Next() {
		key := &journalKey{}
		err := scale.Unmarshal(itr.Key(), key)
		if err != nil {
			return fmt.Errorf("failed to decode journal key %w", err)
		}

		val := itr.Value()

		jr := &journalRecord{}
		err = scale.Unmarshal(val, jr)
		if err != nil {
			return fmt.Errorf("failed to decode journal record block num %d : %w", key.blockNum, err)
		}

		p.addDeathRow(jr, key.blockNum)
	}
	return nil
}

func (p *FullNode) deleteJournalRecord(b chaindb.Batch, key *journalKey) error {
	encKey, err := scale.Marshal(*key)
	if err != nil {
		return err
	}

	err = b.Del(encKey)
	if err != nil {
		return err
	}

	return nil
}

func (p *FullNode) storeLastPrunedIndex(blockNum int64) error {
	encNum, err := scale.Marshal(blockNum)
	if err != nil {
		return err
	}

	err = p.journalDB.Put([]byte(lastPrunedKey), encNum)
	if err != nil {
		return err
	}

	return nil
}

func (p *FullNode) getLastPrunedIndex() (int64, error) {
	val, err := p.journalDB.Get([]byte(lastPrunedKey))
	if err == chaindb.ErrKeyNotFound {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	blockNum := int64(0)
	err = scale.Unmarshal(val, &blockNum)
	if err != nil {
		return 0, err
	}

	return blockNum, nil
}

func (p *FullNode) deleteKeys(b chaindb.Batch, nodesHash map[common.Hash]int64) error {
	for k := range nodesHash {
		err := b.Del(k.ToBytes())
		if err != nil {
			return err
		}
	}

	return nil
}
