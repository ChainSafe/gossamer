package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/pb"
)

//Pruner is an offline tool to prune the stale state with the
// bloom filter, The workflow of pruner is very simple:
// - iterate the storage state, reconstruct the relevant state tries
// - iterate the database, stream all the targeted keys to new DB
type Pruner struct {
	inputDB        *chaindb.BadgerDB
	storageState   *state.StorageState
	blockState     *state.BlockState
	bloom          *stateBloom
	bestBlockHash  common.Hash
	retainBlockNum int64

	prunedDB *badger.DB
}

func newPruner(basePath string, bloomSize uint64, retainBlockNum int64) (*Pruner, error) {
	db, err := loadChainDB(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load DB %w", err)
	}

	base := state.NewBaseState(db)

	bestHash, err := base.LoadBestBlockHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get best block hash: %w", err)
	}

	// load blocktree
	bt := blocktree.NewEmptyBlockTree(db)
	if err = bt.Load(); err != nil {
		return nil, fmt.Errorf("failed to load blocktree: %w", err)
	}

	// create blockState state
	blockState, err := state.NewBlockState(db, bt)
	if err != nil {
		return nil, fmt.Errorf("failed to create block state: %w", err)
	}

	// create bloom filter
	bloom, err := newStateBloomWithSize(bloomSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create new bloom filter of size %d %w", bloomSize, err)
	}

	// load storage state
	storageState, err := state.NewStorageState(db, blockState, trie.NewEmptyTrie())
	if err != nil {
		return nil, fmt.Errorf("failed to create new storage state %w", err)
	}

	return &Pruner{
		inputDB:        db,
		storageState:   storageState,
		blockState:     blockState,
		bloom:          bloom,
		bestBlockHash:  bestHash,
		retainBlockNum: retainBlockNum,
	}, nil
}

// setBloomFilter loads keys with storage prefix of last `retainBlockNum` blocks into the bloom filter
func (p *Pruner) setBloomFilter() error {
	// latest block header
	header, err := p.blockState.GetHeader(p.bestBlockHash)
	if err != nil {
		return err
	}

	latestBlockNum := header.Number.Int64()
	keys := make(map[common.Hash]interface{})

	logger.Info("Latest block number", "num", latestBlockNum)

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

		err = tr.GetDBKey(tr.RootNode(), keys)
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

	fmt.Println("Total keys added in bloom filter", len(keys))

	logger.Info("Total keys added in bloom filter", "keysCount", len(keys))
	return nil
}

func (p *Pruner) prune(inDBPath, pruneDBPath string) error {
	var err error
	p.prunedDB, err = loadBadgerDB(inDBPath)
	if err != nil {
		return fmt.Errorf("failed to load badger DB %w", err)
	}

	defer func() {
		_ = p.prunedDB.Close()
	}()

	if err = p.streamDB(pruneDBPath); err != nil {
		return err
	}

	return nil
}

func (p *Pruner) streamDB(outDir string) error {
	outOptions := badger.DefaultOptions(outDir)
	outOptions.WithInMemory(false)
	outOptions.Dir = outDir

	// Open output DB.
	outDB, err := badger.Open(outOptions)
	if err != nil {
		return fmt.Errorf("cannot open out DB at %s error %w", outDir, err)
	}

	defer func() {
		_ = outDB.Close()
	}()

	writer := outDB.NewStreamWriter()
	if err = writer.Prepare(); err != nil {
		return fmt.Errorf("cannot create stream writer in out DB at %s error %w", outDir, err)
	}

	// Stream contents of DB to the output DB.
	stream := p.prunedDB.NewStream()
	stream.LogPrefix = fmt.Sprintf("Streaming DB to new DB at %s ", outDir)

	stream.ChooseKey = func(item *badger.Item) bool {
		key := string(item.Key())
		if !strings.HasPrefix(key, state.StoragePrefix) {
			return true
		}

		key = strings.TrimPrefix(key, state.StoragePrefix)
		exist := p.bloom.contain([]byte(key))
		return exist
	}

	stream.Send = func(l *pb.KVList) error {
		return writer.Write(l)
	}

	if err = stream.Orchestrate(context.Background()); err != nil {
		return fmt.Errorf("cannot stream DB to out DB at %s error %w", outDir, err)
	}

	if err = writer.Flush(); err != nil {
		return fmt.Errorf("cannot flush writer, error %w", err)
	}

	return nil
}
