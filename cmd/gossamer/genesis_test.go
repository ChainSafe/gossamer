package main

import (
	"bytes"
	"flag"
	"os"
	"reflect"
	"testing"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/polkadb"
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
	if err != nil {
		t.Fatal(err)
	}

	fig, err := getConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := getDataDir(ctx, fig)
	dbSrv, err := polkadb.NewDbService(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	err = dbSrv.Start()
	if err != nil {
		t.Fatal(err)
	}

	defer dbSrv.Stop()

	tdb := &trie.Database{
		Db: dbSrv.StateDB.Db,
	}

	name, err := tdb.Load(common.NodeName)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal([]byte(tmpGenesis.Name), name) {
		t.Fatalf("Fail to get node name: got %s expected %s", name, tmpGenesis.Name)
	}

	id, err := tdb.Load(common.NodeId)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal([]byte(tmpGenesis.Id), id) {
		t.Fatalf("Fail to get node name: got %s expected %s", id, tmpGenesis.Id)
	}

	pid, err := tdb.Load(common.NodeProtocolId)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal([]byte(tmpGenesis.ProtocolId), pid) {
		t.Fatalf("Fail to get node name: got %s expected %s", pid, tmpGenesis.ProtocolId)
	}

	bnodes, err := tdb.Load(common.NodeBootnodes)
	if err != nil {
		t.Fatal(err)
	}

	bnodesArr, err := scale.Decode(bnodes, []string{})
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(bnodesArr, tmpGenesis.Bootnodes) {
		t.Fatalf("Fail to get node name: got %s expected %s", bnodesArr, tmpGenesis.Bootnodes)
	}
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

	err = loadGenesis(context)
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
