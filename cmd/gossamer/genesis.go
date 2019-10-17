package main

import (
	"fmt"

	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/trie"
	"github.com/urfave/cli"
)

func loadGenesis(ctx *cli.Context) (*genesis.GenesisState, error) {
	fig, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}

	// read genesis file
	fp := getGenesisPath(ctx)
	gen, err := genesis.LoadGenesisJsonFile(fp)
	if err != nil {
		return nil, err
	}

	// DB: Create database dir and initialize stateDB and blockDB
	dataDir := getDataDir(ctx, fig)
	dbSrv, err := polkadb.NewDbService(dataDir)
	if err != nil {
		return nil, err
	}

	err = dbSrv.Start()
	if err != nil {
		return nil, err
	}

	// create and load storage trie with initial genesis state
	trieStateDB, err := trie.NewStateDB(dbSrv.StateDB)
	if err != nil {
		return nil, err
	}

	t := trie.NewEmptyTrie(trieStateDB)
	err = t.Load(gen.Genesis.Raw)
	if err != nil {
		return nil, fmt.Errorf("cannot load trie with initial state: %s", err)
	}

	// write initial genesis data to DB
	err = t.WriteToDB()
	if err != nil {
		return nil, err
	}
	err = t.Commit()
	if err != nil {
		return nil, err
	}

	// TODO: create initial p2p config
	return &genesis.GenesisState{
		Name:        gen.Name,
		Id:          gen.Id,
		GenesisTrie: t,
		Db:          dbSrv,
	}, nil
}

// getGenesisPath gets the path to the genesis file
func getGenesisPath(ctx *cli.Context) string {
	if file := ctx.GlobalString(utils.GenesisFlag.Name); file != "" {
		return file
	} else {
		return cfg.DefaultGenesisPath
	}
}
