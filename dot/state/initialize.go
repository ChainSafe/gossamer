// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"
)

// Initialise initialises the genesis state of the DB using the given storage trie.
// The trie should be loaded with the genesis storage state.
// This only needs to be called during genesis initialisation of the node;
// it is not called during normal startup.
func (s *Service) Initialise(gen *genesis.Genesis, header *types.Header, t *trie.Trie) error {
	// get data directory from service
	basepath, err := filepath.Abs(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read basepath: %s", err)
	}

	if err := utils.ClearDatabase(basepath); err != nil {
		return fmt.Errorf("while cleaning database: %w", err)
	}

	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, s.isMemDB)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	s.db = db

	if err = t.WriteDirty(database.NewTable(db, storagePrefix)); err != nil {
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
	rt.Stop()

	// write initial genesis values to database
	if err = s.storeInitialValues(gen.GenesisData(), t); err != nil {
		return fmt.Errorf("failed to write genesis values to database: %s", err)
	}

	tries := NewTries()
	tries.SetTrie(t)

	// create block state from genesis block
	blockState, err := NewBlockStateFromGenesis(db, tries, header, s.Telemetry)
	if err != nil {
		return fmt.Errorf("failed to create block state from genesis: %s", err)
	}

	// create storage state from genesis trie
	storageState, err := NewStorageState(db, blockState, tries)
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

	grandpaState, err := NewGrandpaStateFromGenesis(db, blockState, grandpaAuths, s.Telemetry)
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
		s.Slot = NewSlotState(db)
	} else if err = db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %s", err)
	}

	logger.Infof("block state hash genesis hash: %s", blockState.genesisHash)
	return nil
}

func (s *Service) loadBabeConfigurationFromRuntime(r BabeConfigurer) (*types.BabeConfiguration, error) {
	// load and store initial BABE epoch configuration
	babeCfg, err := r.BabeConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch genesis babe configuration: %w", err)
	}

	if s.BabeThresholdDenominator != 0 {
		babeCfg.C1 = s.BabeThresholdNumerator
		babeCfg.C2 = s.BabeThresholdDenominator
	}

	return babeCfg, nil
}

func loadGrandpaAuthorities(t *trie.Trie) ([]types.GrandpaVoter, error) {
	key := common.MustHexToBytes(genesis.GrandpaAuthoritiesKeyHex)
	authsRaw := t.Get(key)
	if authsRaw == nil {
		return []types.GrandpaVoter{}, nil
	}

	return types.DecodeGrandpaVoters(authsRaw[1:])
}

// storeInitialValues writes initial genesis values to the state database
func (s *Service) storeInitialValues(data *genesis.Data, t *trie.Trie) error {
	// write genesis trie to database
	if err := t.WriteDirty(database.NewTable(s.db, storagePrefix)); err != nil {
		return fmt.Errorf("failed to write trie to database: %s", err)
	}

	// write genesis data to state database
	if err := s.Base.StoreGenesisData(data); err != nil {
		return fmt.Errorf("failed to write genesis data to database: %s", err)
	}

	return nil
}

// CreateGenesisRuntime creates runtime instance form genesis
func (s *Service) CreateGenesisRuntime(t *trie.Trie, gen *genesis.Genesis) (runtime.Instance, error) {
	// load genesis state into database
	genTrie := rtstorage.NewTrieState(t)

	// create genesis runtime
	rtCfg := wazero_runtime.Config{
		LogLvl:  s.logLvl,
		Storage: genTrie,
	}

	r, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis runtime: %w", err)
	}

	return r, nil
}
