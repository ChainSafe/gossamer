// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pruner

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const (
	journalPrefix = "journal"
	lastPrunedKey = "last_pruned"
	pruneInterval = time.Second
)

const (
	// Full pruner mode.
	Full = Mode("full")
	// Archive pruner mode.
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
	RetainedBlocks uint32
}

// Pruner is implemented by FullNode and ArchiveNode.
type Pruner interface {
	StoreJournalRecord(deletedMerkleValues, insertedMerkleValues map[string]struct{},
		blockHash common.Hash, blockNum int64) error
}

// ArchiveNode is a no-op since we don't prune nodes in archive mode.
type ArchiveNode struct{}

// StoreJournalRecord for archive node doesn't do anything.
func (*ArchiveNode) StoreJournalRecord(_, _ map[string]struct{},
	_ common.Hash, _ int64) error {
	return nil
}

type deathRecord struct {
	blockHash                       common.Hash
	deletedMerkleValueToBlockNumber map[string]int64
}

type deathRow []*deathRecord

// FullNode stores state trie diff and allows online state trie pruning
type FullNode struct {
	logger    Logger
	deathList []deathRow
	storageDB chaindb.Database
	journalDB chaindb.Database
	// deathIndex is the mapping from deleted node Merkle value to block number.
	deathIndex map[string]int64
	// pendingNumber is the block number to be pruned.
	// Initial value is set to 1 and is incremented after every block pruning.
	pendingNumber int64
	retainBlocks  uint32
	sync.RWMutex
}

type journalRecord struct {
	// blockHash of the block corresponding to journal record
	blockHash common.Hash
	// Merkle values of nodes inserted in the state trie of the block
	insertedMerkleValues map[string]struct{}
	// Merkle values of nodes deleted from the state trie of the block
	deletedMerkleValues map[string]struct{}
}

type journalKey struct {
	blockNum  int64
	blockHash common.Hash
}

func newJournalRecord(hash common.Hash, insertedMerkleValues,
	deletedMerkleValues map[string]struct{}) *journalRecord {
	return &journalRecord{
		blockHash:            hash,
		insertedMerkleValues: insertedMerkleValues,
		deletedMerkleValues:  deletedMerkleValues,
	}
}

// NewFullNode creates a Pruner for full node.
func NewFullNode(db, storageDB chaindb.Database, retainBlocks uint32, l Logger) (Pruner, error) {
	p := &FullNode{
		deathList:    make([]deathRow, 0),
		deathIndex:   make(map[string]int64),
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
func (p *FullNode) StoreJournalRecord(deletedMerkleValues, insertedMerkleValues map[string]struct{},
	blockHash common.Hash, blockNum int64) error {
	jr := newJournalRecord(blockHash, insertedMerkleValues, deletedMerkleValues)

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

	p.processInsertedKeys(jr.insertedMerkleValues, jr.blockHash)

	// add deleted node Merkle values from journal to death index
	deletedMerkleValueToBlockNumber := make(map[string]int64, len(jr.deletedMerkleValues))
	for k := range jr.deletedMerkleValues {
		p.deathIndex[k] = blockNum
		deletedMerkleValueToBlockNumber[k] = blockNum
	}

	blockIndex := blockNum - p.pendingNumber
	for idx := blockIndex - int64(len(p.deathList)); idx >= 0; idx-- {
		p.deathList = append(p.deathList, deathRow{})
	}

	record := &deathRecord{
		blockHash:                       jr.blockHash,
		deletedMerkleValueToBlockNumber: deletedMerkleValueToBlockNumber,
	}

	// add deathRow to deathList
	p.deathList[blockIndex] = append(p.deathList[blockIndex], record)
}

// Remove re-inserted keys
func (p *FullNode) processInsertedKeys(insertedMerkleValues map[string]struct{}, blockHash common.Hash) {
	for k := range insertedMerkleValues {
		num, ok := p.deathIndex[k]
		if !ok {
			continue
		}
		records := p.deathList[num-p.pendingNumber]
		for _, v := range records {
			if v.blockHash == blockHash {
				delete(v.deletedMerkleValueToBlockNumber, k)
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
		if uint32(len(p.deathList)) <= p.retainBlocks {
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
			err := p.deleteKeys(sdbBatch, record.deletedMerkleValueToBlockNumber)
			if err != nil {
				p.logger.Warnf("failed to prune keys for block number %d: %s", blockNum, err)
				sdbBatch.Reset()
				return
			}

			for k := range record.deletedMerkleValueToBlockNumber {
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

func (*FullNode) deleteJournalRecord(b chaindb.Batch, key *journalKey) error {
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
	if errors.Is(err, chaindb.ErrKeyNotFound) {
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

func (*FullNode) deleteKeys(b chaindb.Batch, deletedMerkleValueToBlockNumber map[string]int64) error {
	for merkleValue := range deletedMerkleValueToBlockNumber {
		err := b.Del([]byte(merkleValue))
		if err != nil {
			return err
		}
	}

	return nil
}
