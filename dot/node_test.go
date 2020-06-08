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
	"bytes"
	"encoding/binary"
	"math/big"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// TestInitNode
func TestInitNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)
}

// TestNodeInitialized
func TestNodeInitialized(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	expected := NodeInitialized(cfg.Global.BasePath, false)
	require.Equal(t, expected, false)

	err := InitNode(cfg)
	require.Nil(t, err)

	expected = NodeInitialized(cfg.Global.BasePath, true)
	require.Equal(t, expected, true)
}

// TestNewNode
func TestNewNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	ks, err := keystore.LoadKeystore("alice")
	require.Nil(t, err)
	require.NotNil(t, ks)

	// TODO: improve dot tests #687
	cfg.Core.Authority = false

	_, err = NewNode(cfg, ks)
	require.Nil(t, err)
}

func TestNewNode_Authority(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	ks, err := keystore.LoadKeystore("alice")
	require.Nil(t, err)
	require.NotNil(t, ks)

	// TODO: improve dot tests #687
	cfg.Core.Authority = true

	_, err = NewNode(cfg, ks)
	require.Nil(t, err)
}

// TestStartNode
func TestStartNode(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()

	err := InitNode(cfg)
	require.Nil(t, err)

	ks, err := keystore.LoadKeystore("alice")
	require.Nil(t, err)
	require.NotNil(t, ks)

	// TODO: improve dot tests #687
	cfg.Core.Authority = false

	node, err := NewNode(cfg, ks)
	require.Nil(t, err)

	go func() {
		err := node.Start()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, uint32(1), atomic.LoadUint32(&node.started))

	node.Stop()
	require.Equal(t, uint32(0), atomic.LoadUint32(&node.started))
}

// TestStopNode

// TODO: improve dot node tests

// TestInitNode_LoadGenesisData
func TestInitNode_LoadGenesisData(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genPath

	err := InitNode(cfg)
	require.Nil(t, err)

	stateSrvc := state.NewService(cfg.Global.BasePath)

	header := &types.Header{
		Number:         big.NewInt(0),
		StateRoot:      trie.EmptyHash,
		ExtrinsicsRoot: trie.EmptyHash,
	}

	gen, err := genesis.NewGenesisFromJSON(genPath)
	require.Nil(t, err)

	err = stateSrvc.Initialize(gen.GenesisData(), header, trie.NewEmptyTrie())
	require.Nil(t, err)

	err = stateSrvc.Start()
	require.Nil(t, err)

	defer func() {
		err = stateSrvc.Stop()
		require.Nil(t, err)
	}()

	gendata, err := state.LoadGenesisData(stateSrvc.DB())
	require.Nil(t, err)

	testGenesis := NewTestGenesis(t)

	expected := &genesis.Data{
		Name:       testGenesis.Name,
		ID:         testGenesis.ID,
		Bootnodes:  common.StringArrayToBytes(testGenesis.Bootnodes),
		ProtocolID: testGenesis.ProtocolID,
	}

	if !reflect.DeepEqual(gendata, expected) {
		t.Fatalf("Fail to get genesis data: got %s expected %s", gendata, expected)
	}

	genesisHeader, err := stateSrvc.Block.BestBlockHeader()
	if err != nil {
		t.Fatal(err)
	}

	stateRoot := genesisHeader.StateRoot
	expectedHeader, err := types.NewHeader(common.NewHash([]byte{0}), big.NewInt(0), stateRoot, trie.EmptyHash, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	if !genesisHeader.Hash().Equal(expectedHeader.Hash()) {
		t.Fatalf("Fail: got %v expected %v", genesisHeader, expectedHeader)
	}
}

// TestInitNode_LoadStorageRoot
func TestInitNode_LoadStorageRoot(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Authority = false
	cfg.Init.Genesis = genPath

	gen, err := genesis.NewGenesisFromJSON(genPath)
	if err != nil {
		t.Fatal(err)
	}

	err = InitNode(cfg)
	require.Nil(t, err)

	ks := keystore.NewKeystore()
	require.NotNil(t, ks)

	node, err := NewNode(cfg, ks)
	require.Nil(t, err)

	if reflect.TypeOf(node) != reflect.TypeOf(&Node{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(node), reflect.TypeOf(&Node{}))
	}

	expected := &trie.Trie{}
	err = expected.Load(gen.GenesisFields().Raw[0])
	require.Nil(t, err)

	expectedRoot, err := expected.Hash()
	require.Nil(t, err)

	mgr := node.Services.Get(&core.Service{})

	var coreSrvc *core.Service
	var ok bool

	if coreSrvc, ok = mgr.(*core.Service); !ok {
		t.Fatal("could not find core service")
	}

	if coreSrvc == nil {
		t.Fatal("core service is nil")
	}

	stateRoot, err := coreSrvc.StorageRoot()
	require.Nil(t, err)

	if !bytes.Equal(expectedRoot[:], stateRoot[:]) {
		t.Fatalf("Fail: got %x expected %x", stateRoot, expectedRoot)
	}
}

func TestInitNode_LoadBalances(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genPath := NewTestGenesisAndRuntime(t)
	require.NotNil(t, genPath)

	defer utils.RemoveTestDir(t)

	cfg.Core.Authority = false
	cfg.Init.Genesis = genPath

	err := InitNode(cfg)
	require.Nil(t, err)

	ks := keystore.NewKeystore()
	require.NotNil(t, ks)

	node, err := NewNode(cfg, ks)
	require.Nil(t, err)

	if reflect.TypeOf(node) != reflect.TypeOf(&Node{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(node), reflect.TypeOf(&Node{}))
	}

	mgr := node.Services.Get(&state.Service{})

	var stateSrv *state.Service
	var ok bool

	if stateSrv, ok = mgr.(*state.Service); !ok {
		t.Fatal("could not find core service")
	}

	if stateSrv == nil {
		t.Fatal("core service is nil")
	}

	kr, _ := keystore.NewSr25519Keyring()
	alice := kr.Alice.Public().(*sr25519.PublicKey).AsBytes()

	bal, err := stateSrv.Storage.GetBalance(alice)
	require.NoError(t, err)

	genbal := "0x0000000000000001"
	balbytes, _ := common.HexToBytes(genbal)
	expected := binary.LittleEndian.Uint64(balbytes)

	require.Equal(t, expected, bal)
}
