package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/pb"
)

// OfflinePruner is a tool to prune the stale state with the help of
// bloom filter, The workflow of Pruner is very simple:
// - iterate the storage state, reconstruct the relevant state tries
// - iterate the database, stream all the targeted keys to new DB
type OfflinePruner struct {
	inputDB        *chaindb.BadgerDB
	storageState   *StorageState
	blockState     *BlockState
	bloom          *bloomState
	bestBlockHash  common.Hash
	retainBlockNum int64

	inputDBPath  string
	prunedDBPath string
}

// NewOfflinePruner creates an instance of OfflinePruner.
func NewOfflinePruner(inputDBPath, prunedDBPath string, bloomSize uint64, retainBlockNum int64) (*OfflinePruner, error) {
	db, err := utils.LoadChainDB(inputDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load DB %w", err)
	}

	// create blockState state
	blockState, err := NewBlockState(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create block state: %w", err)
	}

	bestHash, err := blockState.GetHighestFinalisedHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get best finalised hash: %w", err)
	}

	// create bloom filter
	bloom, err := newBloomState(bloomSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create new bloom filter of size %d %w", bloomSize, err)
	}

	// load storage state
	storageState, err := NewStorageState(db, blockState, trie.NewEmptyTrie(), pruner.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage state %w", err)
	}

	return &OfflinePruner{
		inputDB:        db,
		storageState:   storageState,
		blockState:     blockState,
		bloom:          bloom,
		bestBlockHash:  bestHash,
		retainBlockNum: retainBlockNum,
		prunedDBPath:   prunedDBPath,
		inputDBPath:    inputDBPath,
	}, nil
}

// SetBloomFilter loads keys with storage prefix of last `retainBlockNum` blocks into the bloom filter
func (p *OfflinePruner) SetBloomFilter() error {
	defer p.inputDB.Close() // nolint: errcheck
	finalisedHash, err := p.blockState.GetHighestFinalisedHash()
	if err != nil {
		return fmt.Errorf("failed to get highest finalised hash: %w", err)
	}

	header, err := p.blockState.GetHeader(finalisedHash)
	if err != nil {
		return fmt.Errorf("failed to get highest finalised header: %w", err)
	}

	latestBlockNum := header.Number.Int64()
	keys := make(map[common.Hash]struct{})

	logger.Infof("Latest block number is %d", latestBlockNum)

	if latestBlockNum-p.retainBlockNum <= 0 {
		return fmt.Errorf("not enough block to perform pruning")
	}

	// loop from latest to last `retainBlockNum` blocks
	for blockNum := header.Number.Int64(); blockNum > 0 && blockNum >= latestBlockNum-p.retainBlockNum; {
		var tr *trie.Trie
		tr, err = p.storageState.LoadFromDB(header.StateRoot)
		if err != nil {
			return err
		}

		err = tr.GetNodeHashes(tr.RootNode(), keys)
		if err != nil {
			return err
		}

		// get parent header of current block
		header, err = p.blockState.GetHeader(header.ParentHash)
		if err != nil {
			return err
		}
		blockNum = header.Number.Int64()
	}

	for key := range keys {
		err = p.bloom.put(key.ToBytes())
		if err != nil {
			return err
		}
	}

	logger.Infof("Total keys added in bloom filter: %d", len(keys))
	return nil
}

// Prune starts streaming the data from input db to the pruned db.
func (p *OfflinePruner) Prune() error {
	inputDB, err := utils.LoadBadgerDB(p.inputDBPath)
	if err != nil {
		return fmt.Errorf("failed to load DB %w", err)
	}
	defer inputDB.Close() // nolint: errcheck

	prunedDB, err := utils.LoadBadgerDB(p.prunedDBPath)
	if err != nil {
		return fmt.Errorf("failed to load DB %w", err)
	}
	defer prunedDB.Close() // nolint: errcheck

	writer := prunedDB.NewStreamWriter()
	if err = writer.Prepare(); err != nil {
		return fmt.Errorf("cannot create stream writer in out DB at %s error %w", p.prunedDBPath, err)
	}

	// Stream contents of DB to the output DB.
	stream := inputDB.NewStream()
	stream.LogPrefix = fmt.Sprintf("Streaming DB to new DB at %s ", p.prunedDBPath)

	stream.ChooseKey = func(item *badger.Item) bool {
		key := string(item.Key())
		// All the non storage keys will be streamed to new db.
		if !strings.HasPrefix(key, storagePrefix) {
			return true
		}

		// Only keys present in bloom filter will be streamed to new db
		key = strings.TrimPrefix(key, storagePrefix)
		exist := p.bloom.contain([]byte(key))
		return exist
	}

	stream.Send = func(l *pb.KVList) error {
		return writer.Write(l)
	}

	if err = stream.Orchestrate(context.Background()); err != nil {
		return fmt.Errorf("cannot stream DB to out DB at %s error %w", p.prunedDBPath, err)
	}

	if err = writer.Flush(); err != nil {
		return fmt.Errorf("cannot flush writer, error %w", err)
	}

	return nil
}
