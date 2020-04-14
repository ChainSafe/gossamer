// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package core

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

// TestAuthorityDataKey is the location of authority data in the storage trie
var TestAuthorityDataKey, _ = common.HexToBytes("0xe3b47b6c84c0493481f97c5197d2554f")

// testGenesisHeader is a test block header
var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

// NewTestService creates a new test core service
func NewTestService(t *testing.T, cfg *Config) *Service {
	if cfg == nil {
		rt := runtime.NewTestRuntime(t, runtime.POLKADOT_RUNTIME_c768a7e4c70e)
		cfg = &Config{
			Runtime:         rt,
			IsBabeAuthority: false,
		}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewKeystore()
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.MsgRec == nil {
		cfg.MsgRec = make(chan network.Message, 10)
	}

	if cfg.MsgSend == nil {
		cfg.MsgSend = make(chan network.Message, 10)
	}

	if cfg.SyncChan == nil {
		cfg.SyncChan = make(chan *big.Int, 10)
	}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err := stateSrvc.Initialize(genesisData, testGenesisHeader, trie.NewEmptyTrie())
	require.Nil(t, err)

	err = stateSrvc.Start()
	require.Nil(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	s, err := NewService(cfg)
	require.Nil(t, err)

	return s
}
