package main

import (
	"bytes"
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/p2p"
	"reflect"

	"flag"
	"fmt"
	"github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/polkadb"
	"io/ioutil"
	"os"
	"testing"
	"github.com/urfave/cli"
	log "github.com/ChainSafe/log15"
)

func teardown(tempFile *os.File) {
	if err := os.Remove(tempFile.Name()); err != nil {
		log.Warn("cannot create temp file", err)
	}
}

func createTempConfigFile() (*os.File, *cfg.Config) {
	TestDBConfig := &polkadb.Config{
		DataDir: "chaingang",
	}
	TestP2PConfig := &p2p.Config{
		Port:           cfg.DefaultP2PPort,
		RandSeed:       cfg.DefaultP2PRandSeed,
		BootstrapNodes: []string{
			"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"},
	}
	var TestConfig = &cfg.Config{
		P2pCfg: TestP2PConfig,
		DbCfg:  TestDBConfig,
		RpcCfg: cfg.DefaultRpcConfig,
	}
	tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Crit("Cannot create temporary file", err)
		os.Exit(1)
	}

	f := common.ToTOML(tmpFile.Name(), TestConfig)
	return f, TestConfig
}

func TestGetConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard

	tc := []struct {
		name string
		value string
		usage string
		expected *cfg.Config
	}{
		{"", "", "",cfg.DefaultConfig},
		{"config", tempFile.Name(), "TOML configuration file",cfgClone},
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

		rpc := fmt.Sprintf("%+v", fig.RpcCfg)
		rpcExp := fmt.Sprintf("%+v", c.expected.RpcCfg)

		db := fmt.Sprintf("%+v", fig.DbCfg)
		dbExp := fmt.Sprintf("%+v", c.expected.DbCfg)

		p2p := fmt.Sprintf("%+v", fig.P2pCfg)
		p2pExp := fmt.Sprintf("%+v", c.expected.P2pCfg)

		if !bytes.Equal([]byte(rpc), []byte(rpcExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, rpc, rpcExp)
		}
		if !bytes.Equal([]byte(db), []byte(dbExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, db, dbExp)
		}
		if !bytes.Equal([]byte(p2p), []byte(p2pExp)) {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, p2p, p2pExp)
		}
	}
	defer teardown(tempFile)
}


func TestCommands(t *testing.T) {
	cases := []struct {
		name string
		testArgs               []string
		expectedErr            error
		expextedRes			   string
	}{
		{"dumpConfig",[]string{"dumpConfig"}, nil, cfg.DefaultDataDir()},
	}

	for _, c := range cases {
		app := cli.NewApp()
		app.Writer = ioutil.Discard
		set := flag.NewFlagSet("test", 0)
		set.Parse(c.testArgs)

		context := cli.NewContext(app, set, nil)
		command := dumpConfigCommand

		err := command.Run(context)
		if err != nil {
			t.Fatalf("should have ran dumpConfig command")
		}
	}
}

func TestGetDatabaseDir(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name string
		value string
		usage string
		expected string
	}{
		{"datadir", "test1", "sets database directory","test1"},
		{"config", tempFile.Name(), "TOML configuration file","chaingang"},
	}

	for _, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(app, set, nil)
		dir := getDatabaseDir(context, cfgClone)

		if dir != c.expected {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, dir, c.expected)
		}
	}
}

func TestLoadConfig(t *testing.T) {
}

//func TestCreateP2PService(t *testing.T) {
//	_, cfgClone := createTempConfigFile()
//	srv := createP2PService(cfgClone.P2pCfg)
//
//}

func TestSetBootstrapNodes(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name string
		value string
		usage string
		expected []string
	}{
		{"config", tempFile.Name(), "TOML configuration file",cfgClone.P2pCfg.BootstrapNodes},
		{"bootnodes", "test1", "Comma separated enode URLs for P2P discovery bootstrap",[]string{"test1"}},
	}

	for i, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(nil, set, nil)
		setBootstrapNodes(context, cfgClone.P2pCfg)

		if cfgClone.P2pCfg.BootstrapNodes[i] != c.expected[0] {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, cfgClone.P2pCfg.BootstrapNodes[i], c.expected)
		}
	}
}

func TestSetRpcModules(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name string
		value string
		usage string
		expected []api.Module
	}{
		{"config", tempFile.Name(), "TOML configuration file",[]api.Module{"system"}},
		{"rpcmods", "test1", "API modules to enable via HTTP-RPC, comma separated list",[]api.Module{"test1"}},
	}

	for i, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(nil, set, nil)
		setRpcModules(context, cfgClone.RpcCfg)

		if cfgClone.RpcCfg.Modules[i] != c.expected[0] {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, cfgClone.RpcCfg.Modules[i], c.expected)
		}
	}
}

func TestSetRpcHost(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name string
		value string
		usage string
		expected string
	}{
		{"config", tempFile.Name(), "TOML configuration file","localhost"},
		{"rpchost", "test1", "HTTP-RPC server listening hostname","test1"},
	}

	for _, c := range tc {
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(nil, set, nil)
		setRpcHost(context, cfgClone.RpcCfg)

		if cfgClone.RpcCfg.Host != c.expected {
			t.Fatalf("test failed: %v, got %+v expected %+v", c.name, cfgClone.RpcCfg.Host, c.expected)
		}
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
}
