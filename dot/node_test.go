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

package dot

import (
	"math/big"
	"reflect"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
)

func TestInitNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.NoError(t, err)
}

func TestInitNode_GenesisSpec(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.NoError(t, err)
}

// TestNodeInitialized
func TestNodeInitialized(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	expected := NodeInitialized(cfg.Global.BasePath)
	require.Equal(t, expected, false)

	err := InitNode(cfg)
	require.NoError(t, err)

	expected = NodeInitialized(cfg.Global.BasePath)
	require.Equal(t, expected, true)
}

// TestNewNode
func TestNewNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	err = keystore.LoadKeystore("alice", ks.Gran)
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe)
	require.NoError(t, err)

	cfg.Core.Roles = types.FullNodeRole

	node, err := NewNode(cfg, ks, nil)
	require.NoError(t, err)

	bp := node.Services.Get(&babe.Service{})
	require.NotNil(t, bp)
	fg := node.Services.Get(&grandpa.Service{})
	require.NotNil(t, fg)
}

func TestNewNode_Authority(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	err = keystore.LoadKeystore("alice", ks.Gran)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Gran.Size())
	err = keystore.LoadKeystore("alice", ks.Babe)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Babe.Size())

	cfg.Core.Roles = types.AuthorityRole

	node, err := NewNode(cfg, ks, nil)
	require.NoError(t, err)

	bp := node.Services.Get(&babe.Service{})
	require.NotNil(t, bp)
	fg := node.Services.Get(&grandpa.Service{})
	require.NotNil(t, fg)
}

// TestStartNode
func TestStartNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()
	cfg.Core.GrandpaAuthority = false

	err := InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	err = keystore.LoadKeystore("alice", ks.Gran)
	require.NoError(t, err)
	err = keystore.LoadKeystore("alice", ks.Babe)
	require.NoError(t, err)

	cfg.Core.Roles = types.FullNodeRole

	node, err := NewNode(cfg, ks, nil)
	require.NoError(t, err)

	go func() {
		<-node.started
		node.Stop()
	}()

	err = node.Start()
	require.NoError(t, err)
}

// TestInitNode_LoadGenesisData
func TestInitNode_LoadGenesisData(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genPath
	cfg.Core.GrandpaAuthority = false

	err := InitNode(cfg)
	require.NoError(t, err)

	config := state.Config{
		Path:     cfg.Global.BasePath,
		LogLevel: log.LevelInfo,
	}
	stateSrvc := state.NewService(config)

	gen, err := genesis.NewGenesisFromJSONRaw(genPath)
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)

	err = stateSrvc.Initialise(gen, genesisHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	defer func() {
		err = stateSrvc.Stop()
		require.NoError(t, err)
	}()

	gendata, err := stateSrvc.Base.LoadGenesisData()
	require.NoError(t, err)

	testGenesis := NewTestGenesis(t)

	expected := &genesis.Data{
		Name:       testGenesis.Name,
		ID:         testGenesis.ID,
		Bootnodes:  common.StringArrayToBytes(testGenesis.Bootnodes),
		ProtocolID: testGenesis.ProtocolID,
	}
	require.Equal(t, expected, gendata)

	genesisHeader, err = stateSrvc.Block.BestBlockHeader()
	require.NoError(t, err)

	stateRoot := genesisHeader.StateRoot
	expectedHeader, err := types.NewHeader(common.NewHash([]byte{0}), stateRoot, trie.EmptyHash, big.NewInt(0), types.NewDigest())
	require.NoError(t, err)
	require.Equal(t, expectedHeader.Hash(), genesisHeader.Hash())
}

// TestInitNode_LoadStorageRoot
func TestInitNode_LoadStorageRoot(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genPath

	gen, err := genesis.NewGenesisFromJSONRaw(genPath)
	require.NoError(t, err)

	err = InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())
	sr25519Keyring, _ := keystore.NewSr25519Keyring()
	ks.Babe.Insert(sr25519Keyring.Alice())
	node, err := NewNode(cfg, ks, nil)
	require.NoError(t, err)

	if reflect.TypeOf(node) != reflect.TypeOf(&Node{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(node), reflect.TypeOf(&Node{}))
	}

	expected := &trie.Trie{}
	err = expected.LoadFromMap(gen.GenesisFields().Raw["top"])
	require.NoError(t, err)

	expectedRoot, err := expected.Hash()
	require.NoError(t, err)

	mgr := node.Services.Get(&core.Service{})

	var coreSrvc *core.Service
	var ok bool

	if coreSrvc, ok = mgr.(*core.Service); !ok {
		t.Fatal("could not find core service")
	}
	require.NotNil(t, coreSrvc)

	stateRoot, err := coreSrvc.StorageRoot()
	require.NoError(t, err)
	require.Equal(t, expectedRoot, stateRoot)
}

// balanceKey returns the storage trie key for the balance of the account with the given public key
func balanceKey(t *testing.T, key [32]byte) []byte {
	accKey := append([]byte("balance:"), key[:]...)
	hash, err := common.Blake2bHash(accKey)
	require.NoError(t, err)
	return hash[:]
}

func TestInitNode_LoadBalances(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genPath

	err := InitNode(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	node, err := NewNode(cfg, ks, nil)
	require.NoError(t, err)

	if reflect.TypeOf(node) != reflect.TypeOf(&Node{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(node), reflect.TypeOf(&Node{}))
	}

	mgr := node.Services.Get(&state.Service{})

	var stateSrv *state.Service
	var ok bool

	if stateSrv, ok = mgr.(*state.Service); !ok {
		t.Fatal("could not find core service")
	}
	require.NotNil(t, stateSrv)

	kr, _ := keystore.NewSr25519Keyring()
	alice := kr.Alice().Public().(*sr25519.PublicKey).AsBytes()

	bal, err := stateSrv.Storage.GetStorage(nil, balanceKey(t, alice))
	require.NoError(t, err)

	genbal := "0x0000000000000001"
	expected, _ := common.HexToBytes(genbal)
	require.Equal(t, expected, bal)
}

func TestNode_StopFunc(t *testing.T) {
	testvar := "before"
	stopFunc := func() {
		testvar = "after"
	}

	node := &Node{
		Services: &services.ServiceRegistry{},
		StopFunc: stopFunc,
		wg:       sync.WaitGroup{},
	}
	node.wg.Add(1)

	node.Stop()
	require.Equal(t, testvar, "after")
}

func TestNode_PersistGlobalName_WhenInitialize(t *testing.T) {
	globalName := RandomNodeName()

	cfg := NewTestConfig(t)
	cfg.Global.Name = globalName
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genPath

	err := InitNode(cfg)
	require.NoError(t, err)

	storedName, err := LoadGlobalNodeName(cfg.Global.BasePath)
	require.Nil(t, err)
	require.Equal(t, globalName, storedName)
}
