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
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/ChainSafe/chaindb"
)

// Initialise initialises the genesis state of the DB using the given storage trie. The trie should be loaded with the genesis storage state.
// This only needs to be called during genesis initialisation of the node; it is not called during normal startup.
func (s *Service) Initialise(gen *genesis.Genesis, header *types.Header, t *trie.Trie) error {
	var db chaindb.Database
	cfg := &chaindb.Config{}

	// check database type
	if s.isMemDB {
		cfg.InMemory = true
	}

	// get data directory from service
	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read basepath: %s", err)
	}

	cfg.DataDir = basepath

	// initialise database using data directory
	db, err = chaindb.NewBadgerDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	if err = db.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear database: %s", err)
	}

	if err = t.Store(chaindb.NewTable(db, storagePrefix)); err != nil {
		return fmt.Errorf("failed to write genesis trie to database: %w", err)
	}

	rt, err := s.createGenesisRuntime(t, gen)
	if err != nil {
		return err
	}

	babeCfg, err := s.loadBabeConfigurationFromRuntime(rt)
	if err != nil {
		return err
	}

	// write initial genesis values to database
	if err = s.storeInitialValues(db, gen.GenesisData(), header, t); err != nil {
		return fmt.Errorf("failed to write genesis values to database: %s", err)
	}

	// create and store blockree from genesis block
	bt := blocktree.NewBlockTreeFromRoot(header, db)
	err = bt.Store()
	if err != nil {
		return fmt.Errorf("failed to write blocktree to database: %s", err)
	}

	// create block state from genesis block
	blockState, err := NewBlockStateFromGenesis(db, header)
	if err != nil {
		return fmt.Errorf("failed to create block state from genesis: %s", err)
	}

	// create storage state from genesis trie
	storageState, err := NewStorageState(db, blockState, t)
	if err != nil {
		return fmt.Errorf("failed to create storage state from trie: %s", err)
	}

	epochState, err := NewEpochStateFromGenesis(db, babeCfg)
	if err != nil {
		return fmt.Errorf("failed to create epoch state: %s", err)
	}

	grandpaAuths, err := loadGrandpaAuthorities(t)
	if err != nil {
		return fmt.Errorf("failed to load grandpa authorities: %w", err)
	}

	grandpaState, err := NewGrandpaStateFromGenesis(db, grandpaAuths)
	if err != nil {
		return fmt.Errorf("failed to create grandpa state: %s", err)
	}

	// check database type
	if s.isMemDB {
		// append memory database to state service
		s.db = db

		// append storage state and block state to state service
		s.Storage = storageState
		s.Block = blockState
		s.Epoch = epochState
		s.Grandpa = grandpaState
	} else if err = db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %s", err)
	}

	logger.Info("state", "genesis hash", blockState.genesisHash)
	return nil
}

func (s *Service) loadBabeConfigurationFromRuntime(r runtime.Instance) (*types.BabeConfiguration, error) {
	// load and store initial BABE epoch configuration
	babeCfg, err := r.BabeConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch genesis babe configuration: %w", err)
	}

	r.Stop()

	if s.BabeThresholdDenominator != 0 {
		babeCfg.C1 = s.BabeThresholdNumerator
		babeCfg.C2 = s.BabeThresholdDenominator
	}

	return babeCfg, nil
}

func loadGrandpaAuthorities(t *trie.Trie) ([]*types.GrandpaVoter, error) {
	authsRaw := t.Get(runtime.GrandpaAuthoritiesKey)
	if authsRaw == nil {
		return []*types.GrandpaVoter{}, nil
	}

	r := &bytes.Buffer{}
	_, _ = r.Write(authsRaw[1:])
	return types.DecodeGrandpaVoters(r)
}

// storeInitialValues writes initial genesis values to the state database
func (s *Service) storeInitialValues(db chaindb.Database, data *genesis.Data, header *types.Header, t *trie.Trie) error {
	// write genesis trie to database
	if err := StoreTrie(chaindb.NewTable(db, storagePrefix), t); err != nil {
		return fmt.Errorf("failed to write trie to database: %s", err)
	}

	// write storage hash to database
	if err := StoreLatestStorageHash(db, t.MustHash()); err != nil {
		return fmt.Errorf("failed to write storage hash to database: %s", err)
	}

	// write best block hash to state database
	if err := StoreBestBlockHash(db, header.Hash()); err != nil {
		return fmt.Errorf("failed to write best block hash to database: %s", err)
	}

	// write genesis data to state database
	if err := StoreGenesisData(db, data); err != nil {
		return fmt.Errorf("failed to write genesis data to database: %s", err)
	}

	return nil
}

func (s *Service) createGenesisRuntime(t *trie.Trie, gen *genesis.Genesis) (runtime.Instance, error) {
	// load genesis state into database
	genTrie, err := rtstorage.NewTrieState(t)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate TrieState: %w", err)
	}

	// create genesis runtime
	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genTrie
	rtCfg.LogLvl = s.logLvl

	r, err := wasmer.NewRuntimeFromGenesis(gen, rtCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis runtime: %w", err)
	}

	return r, nil
}
