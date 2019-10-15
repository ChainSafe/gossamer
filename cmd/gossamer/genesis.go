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
	dataDir := getDatabaseDir(ctx, fig)
	dbSrv, err := polkadb.NewDbService(dataDir)
	if err != nil {
		return nil, err
	}

	err = dbSrv.Start()
	if err != nil {
		return nil, err
	}

	trieStateDB, err := trie.NewStateDB(dbSrv.StateDB)
	if err != nil {
		return nil, err
	}

	t := trie.NewEmptyTrie(trieStateDB)
	err = loadTrie(t, gen.Genesis.Raw)
	if err != nil {
		return nil, fmt.Errorf("cannot load trie with initial state: %s", err)
	}

	// write state to DB
	err = commitToDb(t)
	if err != nil {
		return nil, err
	}

	// TODO: load genesis trie and create initial p2p config
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
