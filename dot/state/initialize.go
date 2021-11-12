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
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
)

// Initialise initialises the genesis state of the DB using the given storage trie. The trie should be loaded with the genesis storage state.
// This only needs to be called during genesis initialisation of the node; it is not called during normal startup.
func (s *Service) Initialise(gen *genesis.Genesis, header *types.Header, t *trie.Trie) error {
	// get data directory from service
	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read basepath: %s", err)
	}

	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, s.isMemDB)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	s.db = db

	if err = db.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear database: %s", err)
	}

	if err = t.Store(chaindb.NewTable(db, storagePrefix)); err != nil {
		return fmt.Errorf("failed to write genesis trie to database: %w", err)
	}

	s.Base = NewBaseState(db)

	rt, err := s.CreateGenesisRuntime(t, gen)
	if err != nil {
		return err
	}

	babeCfg, err := s.loadBabeConfigurationFromRuntime(rt)
	if err != nil {
		return err
	}

	// write initial genesis values to database
	if err = s.storeInitialValues(gen.GenesisData(), t); err != nil {
		return fmt.Errorf("failed to write genesis values to database: %s", err)
	}

	// create block state from genesis block
	blockState, err := NewBlockStateFromGenesis(db, header)
	if err != nil {
		return fmt.Errorf("failed to create block state from genesis: %s", err)
	}

	// create storage state from genesis trie
	storageState, err := NewStorageState(db, blockState, t, pruner.Config{})
	if err != nil {
		return fmt.Errorf("failed to create storage state from trie: %s", err)
	}

	epochState, err := NewEpochStateFromGenesis(db, blockState, babeCfg)
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
		// append storage state and block state to state service
		s.Storage = storageState
		s.Block = blockState
		s.Epoch = epochState
		s.Grandpa = grandpaState
	} else if err = db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %s", err)
	}

	logger.Infof("block state hash genesis hash: %s", blockState.genesisHash)
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

func loadGrandpaAuthorities(t *trie.Trie) ([]types.GrandpaVoter, error) {
	authsRaw := t.Get(runtime.GrandpaAuthoritiesKey)
	if authsRaw == nil {
		return []types.GrandpaVoter{}, nil
	}

	return types.DecodeGrandpaVoters(authsRaw[1:])
}

// storeInitialValues writes initial genesis values to the state database
func (s *Service) storeInitialValues(data *genesis.Data, t *trie.Trie) error {
	// write genesis trie to database
	if err := t.Store(chaindb.NewTable(s.db, storagePrefix)); err != nil {
		return fmt.Errorf("failed to write trie to database: %s", err)
	}

	// write genesis data to state database
	if err := s.Base.StoreGenesisData(data); err != nil {
		return fmt.Errorf("failed to write genesis data to database: %s", err)
	}

	if err := s.Base.storePruningData(s.PrunerCfg); err != nil {
		return fmt.Errorf("failed to write pruning data to database: %s", err)
	}

	return nil
}

// CreateGenesisRuntime creates runtime instance form genesis
func (s *Service) CreateGenesisRuntime(t *trie.Trie, gen *genesis.Genesis) (runtime.Instance, error) {
	// load genesis state into database
	genTrie, err := rtstorage.NewTrieState(t)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate TrieState: %w", err)
	}

	// create genesis runtime
	rtCfg := &wasmer.Config{}
	rtCfg.Storage = genTrie
	rtCfg.LogLvl = s.logLvl

	r, err := wasmer.NewRuntimeFromGenesis(rtCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis runtime: %w", err)
	}

	return r, nil
}
