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
	"reflect"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/rpc"

	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

const TestDataDir = "./test_data"

func teardown(tempFile *os.File) {
	if err := os.Remove(tempFile.Name()); err != nil {
		log.Warn("cannot remove temp file", "err", err)
	}
}

func createTempConfigFile() (*os.File, *cfg.Config) {
	testConfig := cfg.DefaultConfig()
	testConfig.GlobalCfg.DataDir = TestDataDir
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

func TestGetConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()
	defer teardown(tempFile)

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
			t.Fatalf("failed to set fig %v", err)
		}

		if !reflect.DeepEqual(fig, c.expected) {
			t.Errorf("\ngot: %+v \nexpected: %+v", fig, c.expected)
		}
	}
}

func TestSetGlobalConfig(t *testing.T) {
	tempFile, cfgClone := createTempConfigFile()

	app := cli.NewApp()
	app.Writer = ioutil.Discard
	tc := []struct {
		name     string
		value    string
		expected cfg.GlobalConfig
	}{
		{"default", "", cfg.GlobalConfig{DataDir: TestDataDir, Verbosity: cfg.DefaultGlobalConfig.Verbosity}},
		{"config", tempFile.Name(), cfg.GlobalConfig{DataDir: TestDataDir, Verbosity: cfg.DefaultGlobalConfig.Verbosity}},
		{"datadir", "test1", cfg.GlobalConfig{DataDir: "test1", Verbosity: cfg.DefaultGlobalConfig.Verbosity}},
	}

	for _, c := range tc {
		c := c // bypass scopelint false positive
		t.Run(c.name, func(t *testing.T) {
			set := flag.NewFlagSet(c.name, 0)
			set.String(c.name, c.value, "")
			context := cli.NewContext(app, set, nil)

			setGlobalConfig(context, &cfgClone.GlobalCfg)

			if !reflect.DeepEqual(cfgClone.GlobalCfg, c.expected) {
				t.Errorf("\ngot: %+v \nexpected: %+v", cfgClone.GlobalCfg, c.expected)
			}
		})
	}
}

func TestCreateP2PService(t *testing.T) {
	srv, _ := createP2PService(cfg.DefaultConfig().P2pCfg)

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
				DataDir:        cfg.DefaultDataDir(),
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
				DataDir:        cfg.DefaultDataDir(),
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
				DataDir:        cfg.DefaultDataDir(),
			},
		},
		{
			"datadir",
			[]string{"datadir"},
			[]interface{}{TestDataDir},
			p2p.Config{
				BootstrapNodes: cfg.DefaultP2PBootstrap,
				Port:           cfg.DefaultP2PPort,
				RandSeed:       cfg.DefaultP2PRandSeed,
				NoBootstrap:    false,
				NoMdns:         false,
				DataDir:        cfgClone.GlobalCfg.DataDir,
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
			// Must call global setup to set data dir
			setGlobalConfig(context, &input.GlobalCfg)
			setP2pConfig(context, input)

			if !reflect.DeepEqual(input.P2pCfg, c.expected) {
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
			setRpcConfig(context, &input.RpcCfg)

			if !reflect.DeepEqual(input.RpcCfg, c.expected) {
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

	for _, c := range tc {
		c := c // bypass scopelint false positive
		set := flag.NewFlagSet(c.name, 0)
		set.String(c.name, c.value, c.usage)
		context := cli.NewContext(nil, set, nil)
		d, fig, _ := makeNode(context, nil)
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
