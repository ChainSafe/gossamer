package main

import (
	"bytes"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/state"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/trie"
	"github.com/urfave/cli"
)

func TestStoreGenesisInfo(t *testing.T) {
	tempFile, _ := createTempConfigFile()
	defer teardown(tempFile)

	genesispath := createTempGenesisFile(t)
	defer os.Remove(genesispath)

	set := flag.NewFlagSet("config", 0)
	set.String("config", tempFile.Name(), "TOML configuration file")
	set.String("genesis", genesispath, "genesis file")
	ctx := cli.NewContext(nil, set, nil)

	err := loadGenesis(ctx)
	require.Nil(t, err)

	currentConfig, err := getConfig(ctx)
	require.Nil(t, err)

	dbSrv := state.NewService(currentConfig.Global.DataDir)

	err = dbSrv.Start()
	require.Nil(t, err)

	defer dbSrv.Stop()

	tdb := &trie.Database{
		Db: dbSrv.Storage.Db.Db,
	}

	gendata, err := tdb.LoadGenesisData()
	require.Nil(t, err)

	expected := &genesis.GenesisData{
		Name:       tmpGenesis.Name,
		ID:         tmpGenesis.ID,
		ProtocolID: tmpGenesis.ProtocolID,
		Bootnodes:  common.StringArrayToBytes(tmpGenesis.Bootnodes),
	}

	if !reflect.DeepEqual(gendata, expected) {
		t.Fatalf("Fail to get genesis data: got %s expected %s", gendata, expected)
	}
}

func TestGenesisStateLoading(t *testing.T) {
	tempFile, _ := createTempConfigFile()
	defer teardown(tempFile)

	genesispath := createTempGenesisFile(t)
	defer os.Remove(genesispath)

	gen, err := genesis.LoadGenesisJsonFile(genesispath)
	require.Nil(t, err)

	set := flag.NewFlagSet("config", 0)
	set.String("config", tempFile.Name(), "TOML configuration file")
	set.String("genesis", genesispath, "genesis file")
	context := cli.NewContext(nil, set, nil)

	err = loadGenesis(context)
	require.Nil(t, err)

	d, _, err := makeNode(context)
	require.Nil(t, err)

	if reflect.TypeOf(d) != reflect.TypeOf(&dot.Dot{}) {
		t.Fatalf("failed to return correct type: got %v expected %v", reflect.TypeOf(d), reflect.TypeOf(&dot.Dot{}))
	}

	expected := &trie.Trie{}
	err = expected.Load(gen.Genesis.Raw)
	require.Nil(t, err)

	expectedRoot, err := expected.Hash()
	require.Nil(t, err)

	mgr := d.Services.Get(&core.Service{})

	stateRoot, err := mgr.(*core.Service).StorageRoot()
	require.Nil(t, err)

	if !bytes.Equal(expectedRoot[:], stateRoot[:]) {
		t.Fatalf("Fail: got %x expected %x", stateRoot, expectedRoot)
	}
}
