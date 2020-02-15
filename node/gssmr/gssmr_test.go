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

package gssmr

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/network"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/state"
	"github.com/ChainSafe/gossamer/tests"
	"github.com/ChainSafe/gossamer/trie"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

const TestDataDir = "./test_data"

const TestProtocolID = "/gossamer/test/0"

var TestBootnodes = []string{
	"/dns4/p2p.cc3-0.kusama.network/tcp/30100/p2p/QmeCit3Nif4VfNqrEJsdYHZGcKzRCnZvGxg6hha1iNj4mk",
	"/dns4/p2p.cc3-1.kusama.network/tcp/30100/p2p/QmchDJtEGiEWf7Ag58HNoTg9jSGzxkSZ23VgmF6xiLKKsZ",
}

var TestGenesis = &genesis.Genesis{
	Name:       "gossamer",
	ID:         "gossamer",
	Bootnodes:  TestBootnodes,
	ProtocolID: TestProtocolID,
	Genesis:    genesis.GenesisFields{},
}

func teardown(tempFile *os.File) {
	if err := os.Remove(tempFile.Name()); err != nil {
		log.Warn("cannot remove temp file", "err", err)
	}
}

func removeTestDataDir() {
	if err := os.RemoveAll(TestDataDir); err != nil {
		log.Warn("cannot remove test data dir", "err", err)
	}
}

func createTempConfigFile() (*os.File, *cfg.Config) {
	testConfig := cfg.DefaultConfig()
	testConfig.Global.DataDir = TestDataDir

	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Crit("Cannot create temporary file", "err", err)
		os.Exit(1)
	}

	f := cfg.ToTOML(tmpFile.Name(), testConfig)
	return f, testConfig
}

// Creates a cli context for a test given a set of flags and values
func createCliContext(description string, flags []string, values []interface{}) (*cli.Context, error) {
	set := flag.NewFlagSet(description, 0)
	for i := range values {
		switch v := values[i].(type) {
		case bool:
			set.Bool(flags[i], v, "")
		case string:
			set.String(flags[i], v, "")
		case uint:
			set.Uint(flags[i], v, "")
		default:
			return nil, fmt.Errorf("unexpected cli value type: %T", values[i])
		}
	}
	context := cli.NewContext(nil, set, nil)
	return context, nil
}

func createTempGenesisFile(t *testing.T) string {
	_ = runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	testRuntimeFilePath := tests.GetAbsolutePath(tests.POLKADOT_RUNTIME_FP)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.Nil(t, err)

	testBytes, err := ioutil.ReadFile(fp)
	require.Nil(t, err)

	testHex := hex.EncodeToString(testBytes)
	testRaw := [2]map[string]string{}
	testRaw[0] = map[string]string{"0x3a636f6465": "0x" + testHex}
	TestGenesis.Genesis = genesis.GenesisFields{Raw: testRaw}

	// Create temp file
	file, err := ioutil.TempFile(os.TempDir(), "genesis-test")
	require.Nil(t, err)

	// Grab json encoded bytes
	bz, err := json.Marshal(TestGenesis)
	require.Nil(t, err)

	// Write to temp file
	_, err = file.Write(bz)
	require.Nil(t, err)

	return file.Name()
}

// Creates a Node with default configurations. Does not include RPC server.
func createTestNode(t *testing.T, testDir string) *Node {
	var services []services.Service

	// Network
	networkCfg := &network.Config{
		BlockState:   &state.BlockState{},   // required
		NetworkState: &state.NetworkState{}, // required
		StorageState: &state.StorageState{}, // required
		DataDir:      testDir,               // default "~/.gossamer"
		Roles:        1,                     // required
		RandSeed:     1,                     // default 0
	}
	networkSrvc, err := network.NewService(networkCfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	services = append(services, networkSrvc)

	// DB
	dbSrv := state.NewService(testDir)
	err = dbSrv.Initialize(&types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}, trie.NewEmptyTrie(nil))
	if err != nil {
		t.Fatal(err)
	}
	services = append(services, dbSrv)

	// API
	apiSrvc := api.NewAPIService(networkSrvc, nil)
	services = append(services, apiSrvc)

	return NewNode("gssmr", services, nil)
}

func TestNode_Start(t *testing.T) {
	testDir := path.Join(os.TempDir(), "gssmr-test")
	defer os.RemoveAll(testDir)

	availableServices := [...]services.Service{
		&network.Service{},
		&api.Service{},
		&state.Service{},
	}

	gssmr := createTestNode(t, testDir)

	go gssmr.Start()

	// Wait until gssmr.Start() is finished
	<-gssmr.IsStarted

	for _, srvc := range availableServices {
		s := gssmr.Services.Get(srvc)
		if s == nil {
			t.Fatalf("error getting service: %T", srvc)
		}
	}

	gssmr.Stop()

	// Wait for everything to finish
	<-gssmr.stop
}

func TestCreateNetworkService(t *testing.T) {
	stateSrv := state.NewService(TestDataDir)
	srv, _, _ := createNetworkService(cfg.DefaultConfig(), &genesis.GenesisData{}, stateSrv)
	require.NotNil(t, srv, "failed to create network service")
}
