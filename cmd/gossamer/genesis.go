package main

import (
	"fmt"

	"github.com/ChainSafe/gossamer/state"

	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/trie"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

func loadGenesis(ctx *cli.Context) error {
	fig, err := getConfig(ctx)
	if err != nil {
		return err
	}

	fp := getGenesisPath(ctx)

	dataDir := fig.Global.DataDir
	if ctx.String(utils.DataDirFlag.Name) != "" {
		dataDir = ctx.String(utils.DataDirFlag.Name)
	}

	log.Debug("Loading genesis", "genesisfile", fp, "datadir", dataDir)

	// read genesis configuration file
	gen, err := genesis.LoadGenesisJSONFile(fp)
	if err != nil {
		return err
	}

	log.Info("🕸\t Initializing node", "name", gen.Name, "id", gen.Id, "protocolID", gen.ProtocolId, "bootnodes", gen.Bootnodes)

	// Create service, initialize stateDB and blockDB
	stateSrv := state.NewService(dataDir)

	err = stateSrv.Start()
	if err != nil {
		return err
	}

	defer func() {
		err = stateSrv.Stop()
		if err != nil {
			log.Error("error stopping database service")
		}
	}()

	tdb := &trie.Database{
		Db: stateSrv.Storage.Db.Db,
	}

	// create and load storage trie with initial genesis state
	t := trie.NewEmptyTrie(tdb)

	err = t.Load(gen.GenesisFields().Raw[0])
	if err != nil {
		return fmt.Errorf("cannot load trie with initial state: %s", err)
	}

	// write initial genesis data to DB
	err = t.StoreInDB()
	if err != nil {
		return fmt.Errorf("cannot store genesis data in db: %s", err)
	}

	err = t.StoreHash()
	if err != nil {
		return fmt.Errorf("cannot store genesis hash in db: %s", err)
	}

	// store node name, ID, p2p protocol, bootnodes in DB
	return t.Db().StoreGenesisData(gen.GenesisData())
}

// getGenesisPath gets the path to the genesis file
func getGenesisPath(ctx *cli.Context) string {
	// Check local string genesis flags first
	if file := ctx.String(utils.GenesisFlag.Name); file != "" {
		return file
	} else if file := ctx.GlobalString(utils.GenesisFlag.Name); file != "" {
		return file
	} else {
		return cfg.DefaultGenesisPath
	}
}
