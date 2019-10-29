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

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/trie"

	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

const TestDataDir = "./test_data"

func teardown(tempFile *os.File) {
	if err := os.Remove(tempFile.Name()); err != nil {
		log.Warn("cannot create temp file", err)
	}
	if err := os.RemoveAll("./test_data"); err != nil {
		log.Warn("removal of temp directory bin failed", "err", err)
	}
}

func createTempConfigFile() (*os.File, *cfg.Config) {
	testConfig := cfg.DefaultConfig()
	testConfig.DbCfg.DataDir = TestDataDir
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Crit("Cannot create temporary file", err)
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

const TESTS_FP string = "../../runtime/test_wasm.wasm"
const TEST_WASM_URL string = "https://github.com/ChainSafe/gossamer-test-wasm/blob/c0ff6e519676affd727a45fe605bc7c84a0a536d/target/wasm32-unknown-unknown/release/test_wasm.wasm?raw=true"

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// getTestBlob checks if the test wasm file exists and if not, it fetches it from github
func getTestBlob() (n int64, err error) {
	if Exists(TESTS_FP) {
		return 0, nil
	}

	out, err := os.Create(TESTS_FP)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	resp, err := http.Get(TEST_WASM_URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

func createTempGenesisFile(t *testing.T) string {
	_, err := getTestBlob()
	if err != nil {
		t.Fatal(err)
	}

	fp, err := filepath.Abs(TESTS_FP)
	if err != nil {
		t.Fatal(err)
	}

	testbytes, err := ioutil.ReadFile(fp)
	if err != nil {
		t.Fatal(err)
	}

	testhex := hex.EncodeToString(testbytes)

	tmp := &genesis.Genesis{
		Name:       "gossamer",
		Id:         "gossamer",
		Bootnodes:  []string{"/ip4/104.211.54.233/tcp/30363/p2p/16Uiu2HAmFWPUx45xYYeCpAryQbvU3dY8PWGdMwS2tLm1dB1CsmCj"},
		ProtocolId: "gossamer",
		Genesis: genesis.GenesisFields{
			Raw: map[string]string{"0x3a636f6465": "0x" + testhex},
		},
	}

	// Create temp file
	file, err := ioutil.TempFile("", "genesis-test")
	if err != nil {
		t.Fatal(err)
	}

	// Grab json encoded bytes
	bz, err := json.Marshal(tmp)
	if err != nil {
		t.Fatal(err)
	}

	// Write to temp file
	_, err = file.Write(bz)
	if err != nil {
		t.Fatal(err)
	}

	return file.Name()
}

func TestGetConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard

	tc := []struct {
		name     string
		value    string
		usage    string
		expected *cfg.Config
	}{
		{"", "", "", cfg.DefaultConfig()},
		{"config", tempFile.Name(), "TOML configuration file", cfgClone},
	}

	for _, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(app, set, nil)

		fig, err := getConfig(context)
		if err != nil {
			teardown(tempFile)
			t.Fatalf("failed to set fig %v", err)
		}

		r := fmt.Sprintf("%+v", fig.RpcCfg)
		rpcExp := fmt.Sprintf("%+v", c.expected.RpcCfg)

		db := fmt.Sprintf("%+v", fig.DbCfg)
		dbExp := fmt.Sprintf("%+v", c.expected.DbCfg)

		peer := fmt.Sprintf("%+v", fig.P2pCfg)
		p2pExp := fmt.Sprintf("%+v", c.expected.P2pCfg)

		if !bytes.Equal([]byte(r), []byte(rpcExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, r, rpcExp)
		}
		if !bytes.Equal([]byte(db), []byte(dbExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, db, dbExp)
		}
		if !bytes.Equal([]byte(peer), []byte(p2pExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, peer, p2pExp)
		}
	}
	defer teardown(tempFile)
}

func TestGetDatabaseDir(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name     string
		value    string
		usage    string
		expected string
	}{
		{"", "", "", cfg.DefaultDBConfig.DataDir},
		{"config", tempFile.Name(), "TOML configuration file", TestDataDir},
		{"datadir", "test1", "sets database directory", "test1"},
	}

	for i, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(app, set, nil)
		if i == 0 {
			cfgClone.DbCfg.DataDir = ""
		} else {
			cfgClone.DbCfg.DataDir = TestDataDir
		}
		dir := getDataDir(context, cfgClone)

		if dir != c.expected {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, dir, c.expected)
		}
	}
}

func TestCreateP2PService(t *testing.T) {
	_, cfgClone := createTempConfigFile()
	srv, _ := createP2PService(cfgClone.P2pCfg)

	if srv == nil {
		t.Fatalf("failed to create p2p service")
	}
}

func TestSetP2pConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    p2p.Config
	}{
		{
			"config file",
			[]string{"config"},
			[]interface{}{tempFile.Name()},
			cfgClone.P2pCfg,
		},
		{
			"no bootstrap, no mdns",
			[]string{"nobootstrap", "nomdns"},
			[]interface{}{true, true},
			p2p.Config{
				BootstrapNodes: cfg.DefaultP2PBootstrap,
				Port:           cfg.DefaultP2PPort,
				RandSeed:       cfg.DefaultP2PRandSeed,
				NoBootstrap:    true,
				NoMdns:         true,
			},
		},
		{
			"bootstrap nodes",
			[]string{"bootnodes"},
			[]interface{}{"1234,5678"},
			p2p.Config{
				BootstrapNodes: []string{"1234", "5678"},
				Port:           cfg.DefaultP2PPort,
				RandSeed:       cfg.DefaultP2PRandSeed,
				NoBootstrap:    false,
				NoMdns:         false,
			},
		},
		{
			"port",
			[]string{"p2pport"},
			[]interface{}{uint(1337)},
			p2p.Config{
				BootstrapNodes: cfg.DefaultP2PBootstrap,
				Port:           1337,
				RandSeed:       cfg.DefaultP2PRandSeed,
				NoBootstrap:    false,
				NoMdns:         false,
			},
		},
	}

	for _, c := range tc {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			context, err := createCliContext(c.description, c.flags, c.values)
			if err != nil {
				t.Fatal(err)
			}

			input := cfg.DefaultConfig()
			res := setP2pConfig(context, input.P2pCfg)

			if !reflect.DeepEqual(res, c.expected) {
				t.Fatalf("\ngot %+v\nexpected %+v", input.P2pCfg, c.expected)
			}
		})
	}
}

func TestSetRpcConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    rpc.Config
	}{
		{
			"config file",
			[]string{"config"},
			[]interface{}{tempFile.Name()},
			cfgClone.RpcCfg,
		},
		{
			"host and port",
			[]string{"rpchost", "rpcport"},
			[]interface{}{"someHost", uint(1337)},
			rpc.Config{
				Port:    1337,
				Host:    "someHost",
				Modules: cfg.DefaultRpcModules,
			},
		},
		{
			"modules",
			[]string{"rpcmods"},
			[]interface{}{"system,state"},
			rpc.Config{
				Port:    cfg.DefaultRpcHttpPort,
				Host:    cfg.DefaultRpcHttpHost,
				Modules: []api.Module{"system", "state"},
			},
		},
	}

	for _, c := range tc {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			context, err := createCliContext(c.description, c.flags, c.values)
			if err != nil {
				t.Fatal(err)
			}

			input := cfg.DefaultConfig()
			res := setRpcConfig(context, input.RpcCfg)

			if !reflect.DeepEqual(res, c.expected) {
				t.Fatalf("\ngot %+v\nexpected %+v", input.RpcCfg, c.expected)
			}
		})
	}
}

func TestStrToMods(t *testing.T) {
	strs := []string{"test1", "test2"}
	mods := strToMods(strs)
	rv := reflect.ValueOf(mods)
	if rv.Kind() == reflect.Ptr {
		t.Fatalf("test failed: got %v expected %v", mods, &[]api.Module{})
	}
}

func TestMakeNode(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name     string
		value    string
		usage    string
		expected *cfg.Config
	}{
		{"config", tempFile.Name(), "TOML configuration file", cfgClone},
	}

	genesispath := createTempGenesisFile(t)
	defer os.Remove(genesispath)

	for _, c := range tc {
		c := c // bypass scopelint false positive
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		set.String("genesis", genesispath, "genesis file")
		context := cli.NewContext(nil, set, nil)

		_, err := loadGenesis(context)
		if err != nil {
			t.Fatal(err)
		}

		d, fig, err := makeNode(context)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.TypeOf(d) != reflect.TypeOf(&dot.Dot{}) {
			t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(d), reflect.TypeOf(&dot.Dot{}))
		}
		if reflect.TypeOf(d.Services) != reflect.TypeOf(&services.ServiceRegistry{}) {
			t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(d.Services), reflect.TypeOf(&services.ServiceRegistry{}))
		}
		if reflect.TypeOf(d.Rpc) != reflect.TypeOf(&rpc.HttpServer{}) {
			t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(d.Rpc), reflect.TypeOf(&rpc.HttpServer{}))
		}
		if reflect.TypeOf(fig) != reflect.TypeOf(&cfg.Config{}) {
			t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(fig), reflect.TypeOf(&cfg.Config{}))
		}
	}
	defer teardown(tempFile)
}

func TestCommands(t *testing.T) {
	tempFile, _ := createTempConfigFile()

	tc := []struct {
		description string
		flags       []string
		values      []interface{}
	}{
		{"from config file",
			[]string{"config"},
			[]interface{}{tempFile.Name()}},
	}

	for _, c := range tc {
		c := c // bypass scopelint false positive

		app := cli.NewApp()
		app.Writer = ioutil.Discard

		context, err := createCliContext(c.description, c.flags, c.values)
		if err != nil {
			t.Fatal(err)
		}

		command := dumpConfigCommand

		err = command.Run(context)
		if err != nil {
			t.Fatalf("should have ran dumpConfig command. err: %s", err)
		}
	}
	defer teardown(tempFile)
}

func TestGenesisStateLoading(t *testing.T) {
	tempFile, _ := createTempConfigFile()
	defer teardown(tempFile)

	genesispath := createTempGenesisFile(t)
	defer os.Remove(genesispath)

	gen, err := genesis.LoadGenesisJsonFile(genesispath)
	if err != nil {
		t.Fatal(err)
	}

	set := flag.NewFlagSet("config", 0)
	set.String("config", tempFile.Name(), "TOML configuration file")
	set.String("genesis", genesispath, "genesis file")
	context := cli.NewContext(nil, set, nil)

	_, err = loadGenesis(context)
	if err != nil {
		t.Fatal(err)
	}

	d, _, err := makeNode(context)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.TypeOf(d) != reflect.TypeOf(&dot.Dot{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(d), reflect.TypeOf(&dot.Dot{}))
	}

	expected := &trie.Trie{}
	err = expected.Load(gen.Genesis.Raw)
	if err != nil {
		t.Fatal(err)
	}

	expectedRoot, err := expected.Hash()
	if err != nil {
		t.Fatal(err)
	}

	mgr := d.Services.Get(&core.Service{})

	stateRoot, err := mgr.(*core.Service).StorageRoot()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expectedRoot[:], stateRoot[:]) {
		t.Fatalf("Fail: got %x expected %x", stateRoot, expectedRoot)
	}
}
