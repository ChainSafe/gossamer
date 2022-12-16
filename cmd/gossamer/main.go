// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"errors"
	"fmt"
	"os"
	_ "time/tzdata"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/urfave/cli"

	_ "github.com/breml/rootcerts"
)

const (
	accountCommandName       = "account"
	exportCommandName        = "export"
	initCommandName          = "init"
	buildSpecCommandName     = "build-spec"
	importRuntimeCommandName = "import-runtime"
	importStateCommandName   = "import-state"
	pruningStateCommandName  = "prune-state"
)

// app is the cli application
var app = cli.NewApp()
var logger Logger = log.NewFromGlobal(log.AddContext("pkg", "cmd"))

var (
	// exportCommand defines the "export" subcommand (ie, `gossamer export`)
	exportCommand = cli.Command{
		Action:    FixFlagOrder(exportAction),
		Name:      exportCommandName,
		Usage:     "Export configuration values to TOML configuration file",
		ArgsUsage: "",
		Flags:     ExportFlags,
		Category:  "EXPORT",
		Description: "The export command exports configuration values from " +
			"the command flags to a TOML configuration file.\n" +
			"\tUsage: gossamer export --config chain/test/config.toml --basepath ~/.gossamer/test",
	}
	// initCommand defines the "init" subcommand (ie, `gossamer init`)
	initCommand = cli.Command{
		Action:    FixFlagOrder(initAction),
		Name:      initCommandName,
		Usage:     "Initialise node databases and load genesis data to state",
		ArgsUsage: "",
		Flags:     InitFlags,
		Category:  "INIT",
		Description: "The init command initialises the node databases and " +
			"loads the genesis data from the genesis file to state.\n" +
			"\tUsage: gossamer init --genesis genesis.json",
	}
	// accountCommand defines the "account" subcommand (ie, `gossamer account`)
	accountCommand = cli.Command{
		Action:   FixFlagOrder(accountAction),
		Name:     accountCommandName,
		Usage:    "Create and manage node keystore accounts",
		Flags:    AccountFlags,
		Category: "ACCOUNT",
		Description: "The account command is used to manage the gossamer keystore.\n" +
			"\tTo generate a new sr25519 account: gossamer account --generate\n" +
			"\tTo generate a new ed25519 account: gossamer account --generate --ed25519\n" +
			"\tTo generate a new secp256k1 account: gossamer account --generate --secp256k1\n" +
			"\tTo import a keystore file: gossamer account --import=path/to/file\n" +
			"\tTo list keys: gossamer account --list",
	}
	// buildSpecCommand creates a raw genesis file from a human readable genesis file.
	buildSpecCommand = cli.Command{
		Action:    FixFlagOrder(buildSpecAction),
		Name:      buildSpecCommandName,
		Usage:     "Generates genesis JSON data, and can convert to raw genesis data",
		ArgsUsage: "",
		Flags:     BuildSpecFlags,
		Category:  "BUILD-SPEC",
		Description: "The build-spec command outputs current genesis JSON data.\n" +
			"\tUsage: gossamer build-spec\n" +
			"\tTo generate raw genesis file from default: " +
			"gossamer build-spec --raw --output genesis.json" +
			"\tTo generate raw genesis file from specific genesis file: " +
			"gossamer build-spec --raw --genesis genesis-spec.json --output genesis.json",
	}

	// importRuntime generates a genesis file given a .wasm runtime binary.
	importRuntimeCommand = cli.Command{
		Action:    FixFlagOrder(importRuntimeAction),
		Name:      importRuntimeCommandName,
		Usage:     "Generates a genesis file given a .wasm runtime binary",
		ArgsUsage: "",
		Flags:     RootFlags,
		Category:  "IMPORT-RUNTIME",
		Description: "The import-runtime command generates a genesis file given a .wasm runtime binary.\n" +
			"\tUsage: gossamer import-runtime runtime.wasm > genesis.json\n",
	}

	importStateCommand = cli.Command{
		Action:    FixFlagOrder(importStateAction),
		Name:      importStateCommandName,
		Usage:     "Import state from a JSON file and set it as the chain head state",
		ArgsUsage: "",
		Flags:     ImportStateFlags,
		Category:  "IMPORT-STATE",
		Description: "The import-state command allows a JSON file containing a given state " +
			"in the form of key-value pairs to be imported.\n" +
			"Input can be generated by using the RPC function state_getPairs.\n" +
			"\tUsage: gossamer import-state --state state.json --header header.json --first-slot <first slot of network>\n",
	}

	pruningCommand = cli.Command{
		Action:    FixFlagOrder(pruneState),
		Name:      pruningStateCommandName,
		Usage:     "Prune state will prune the state trie",
		ArgsUsage: "",
		Flags:     PruningFlags,
		Description: `prune-state <retain-blocks> will prune historical state data.
		All trie nodes that do not belong to the specified version state will be deleted from the database.

		The default pruning target is the HEAD-256 state`,
	}
)

// init initialises the cli application
func init() {
	app.Action = gossamerAction
	app.Copyright = "Copyright 2019 ChainSafe Systems Authors"
	app.Name = "gossamer"
	app.Usage = "Official gossamer command-line interface"
	app.Author = "ChainSafe Systems 2019"
	app.Version = "0.3.2"
	app.Commands = []cli.Command{
		exportCommand,
		initCommand,
		accountCommand,
		buildSpecCommand,
		importRuntimeCommand,
		importStateCommand,
		pruningCommand,
	}
	app.Flags = RootFlags
}

// main runs the cli application
func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func importStateAction(ctx *cli.Context) error {
	var (
		stateFP, headerFP string
		firstSlot         int
	)

	if stateFP = ctx.String(StateFlag.Name); stateFP == "" {
		return errors.New("must provide argument to --state")
	}

	if headerFP = ctx.String(HeaderFlag.Name); headerFP == "" {
		return errors.New("must provide argument to --header")
	}

	if firstSlot = ctx.Int(FirstSlotFlag.Name); firstSlot == 0 {
		return errors.New("must provide argument to --first-slot")
	}

	cfg, err := createImportStateConfig(ctx)
	if err != nil {
		logger.Errorf("failed to create node configuration: %s", err)
		return err
	}
	cfg.Global.BasePath = utils.ExpandDir(cfg.Global.BasePath)

	return dot.ImportState(cfg.Global.BasePath, stateFP, headerFP, uint64(firstSlot))
}

// importRuntimeAction generates a genesis file given a .wasm runtime binary.
func importRuntimeAction(ctx *cli.Context) error {
	arguments := ctx.Args()
	if len(arguments) == 0 {
		return fmt.Errorf("no args provided, please provide wasm file")
	}

	fp := arguments[0]
	out, err := createGenesisWithRuntime(fp)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

// gossamerAction is the root action for the gossamer command, creates a node
// configuration, loads the keystore, initialises the node if not initialised,
// then creates and starts the node and node services
func gossamerAction(ctx *cli.Context) error {
	// check for unknown command arguments
	if arguments := ctx.Args(); len(arguments) > 0 {
		return fmt.Errorf("failed to read command argument: %q", arguments[0])
	}

	// setup gossamer logger
	lvl, err := setupLogger(ctx)
	if err != nil {
		logger.Errorf("failed to setup logger: %s", err)
		return err
	}

	// create new dot configuration (the dot configuration is created within the
	// cli application from the flag values provided)
	cfg, err := createDotConfig(ctx)
	if err != nil {
		logger.Errorf("failed to create node configuration: %s", err)
		return err
	}

	cfg.Global.LogLvl = lvl

	// expand data directory and update node configuration (performed separately
	// from createDotConfig because dot config should not include expanded path)
	cfg.Global.BasePath = utils.ExpandDir(cfg.Global.BasePath)

	if !dot.IsNodeInitialised(cfg.Global.BasePath) {
		// initialise node (initialise state database and load genesis data)
		err = dot.InitNode(cfg)
		if err != nil {
			logger.Errorf("failed to initialise node: %s", err)
			return err
		}
	}

	// ensure configuration matches genesis data stored during node initialization
	// but do not overwrite configuration if the corresponding flag value is set
	err = updateDotConfigFromGenesisData(ctx, cfg)
	if err != nil {
		logger.Errorf("failed to update config from genesis data: %s", err)
		return err
	}

	ks := keystore.NewGlobalKeystore()

	if cfg.Account.Key != "" {
		err = loadBuiltInTestKeys(cfg.Account.Key, *ks)
		if err != nil {
			return fmt.Errorf("loading built-in test keys: %s", err)
		}
	}

	// load user keys if specified
	err = unlockKeystore(ks.Acco, cfg.Global.BasePath, cfg.Account.Unlock, ctx.String(PasswordFlag.Name))
	if err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	err = unlockKeystore(ks.Babe, cfg.Global.BasePath, cfg.Account.Unlock, ctx.String(PasswordFlag.Name))
	if err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	err = unlockKeystore(ks.Gran, cfg.Global.BasePath, cfg.Account.Unlock, ctx.String(PasswordFlag.Name))
	if err != nil {
		logger.Errorf("failed to unlock keystore: %s", err)
		return err
	}

	node, err := dot.NewNode(cfg, ks)
	if err != nil {
		logger.Errorf("failed to create node services: %s", err)
		return err
	}

	logger.Info("starting node " + node.Name + "...")

	// start node
	err = node.Start()
	if err != nil {
		return err
	}

	return nil
}

func loadBuiltInTestKeys(accountKey string, ks keystore.GlobalKeystore) (err error) {
	sr25519keyRing, err := keystore.NewSr25519Keyring()
	if err != nil {
		return fmt.Errorf("creating sr22519 keyring: %s", err)
	}

	ed25519keyRing, err := keystore.NewEd25519Keyring()
	if err != nil {
		return fmt.Errorf("creating ed25519 keyring: %s", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Acco, sr25519keyRing)
	if err != nil {
		return fmt.Errorf("loading account keystore: %w", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Babe, sr25519keyRing)
	if err != nil {
		return fmt.Errorf("loading babe keystore: %w", err)
	}

	err = keystore.LoadKeystore(accountKey, ks.Gran, ed25519keyRing)
	if err != nil {
		return fmt.Errorf("loading grandpa keystore: %w", err)
	}

	return nil
}

// initAction is the action for the "init" subcommand, initialises the trie and
// state databases and loads initial state from the configured genesis file
func initAction(ctx *cli.Context) error {
	lvl, err := setupLogger(ctx)
	if err != nil {
		logger.Errorf("failed to setup logger: %s", err)
		return err
	}

	cfg, err := createInitConfig(ctx)
	if err != nil {
		logger.Errorf("failed to create node configuration: %s", err)
		return err
	}

	cfg.Global.LogLvl = lvl

	// expand data directory and update node configuration (performed separately
	// from createDotConfig because dot config should not include expanded path)
	cfg.Global.BasePath = utils.ExpandDir(cfg.Global.BasePath)
	// check if node has been initialised (expected false - no warning log)
	if dot.IsNodeInitialised(cfg.Global.BasePath) {
		// use --force value to force initialise the node
		force := ctx.Bool(ForceFlag.Name)

		// prompt user to confirm reinitialization
		if force || confirmMessage("Are you sure you want to reinitialise the node? [Y/n]") {
			logger.Info("reinitialising node at base path " + cfg.Global.BasePath + "...")
		} else {
			logger.Warn("exiting without reinitialising the node at base path " + cfg.Global.BasePath + "...")
			return nil // exit if reinitialization is not confirmed
		}
	}

	// initialise node (initialise state database and load genesis data)
	err = dot.InitNode(cfg)
	if err != nil {
		logger.Errorf("failed to initialise node: %s", err)
		return err
	}

	return nil
}

func buildSpecAction(ctx *cli.Context) error {
	// set logger to critical, so output only contains genesis data
	err := ctx.Set("log", "critical")
	if err != nil {
		return err
	}
	_, err = setupLogger(ctx)
	if err != nil {
		return err
	}

	var bs *dot.BuildSpec

	if genesis := ctx.String(GenesisSpecFlag.Name); genesis != "" {
		bspec, e := dot.BuildFromGenesis(genesis, 0)
		if e != nil {
			return e
		}
		bs = bspec
	} else {
		cfg, e := createBuildSpecConfig(ctx)
		if e != nil {
			return e
		}
		// expand data directory and update node configuration (performed separately
		// from createDotConfig because dot config should not include expanded path)
		cfg.Global.BasePath = utils.ExpandDir(cfg.Global.BasePath)

		bspec, e := dot.BuildFromDB(cfg.Global.BasePath)
		if e != nil {
			return fmt.Errorf("error building spec from database, "+
				"init must be run before build-spec or "+
				"run build-spec with --genesis flag Error %s", e)
		}
		bs = bspec
	}

	if bs == nil {
		return fmt.Errorf("error building genesis")
	}

	var res []byte

	if ctx.Bool(RawFlag.Name) {
		res, err = bs.ToJSONRaw()
	} else {
		res, err = bs.ToJSON()
	}

	if err != nil {
		return err
	}

	if outputPath := ctx.String(OutputSpecFlag.Name); outputPath != "" {
		err = dot.WriteGenesisSpecFile(res, outputPath)
		if err != nil {
			return fmt.Errorf("cannot write genesis spec file: %w", err)
		}
	} else {
		fmt.Printf("%s\n", res)
	}

	return nil
}

func pruneState(ctx *cli.Context) error {
	tomlCfg, _, err := setupConfigFromChain(ctx)
	if err != nil {
		logger.Errorf("failed to load chain configuration: %s", err)
		return err
	}

	inputDBPath := tomlCfg.Global.BasePath
	prunedDBPath := ctx.GlobalString(DBPathFlag.Name)
	if prunedDBPath == "" {
		return fmt.Errorf("path not specified for badger db")
	}

	bloomSize := ctx.GlobalUint64(BloomFilterSizeFlag.Name)

	const uint32Max = ^uint32(0)
	flagValue := ctx.GlobalUint64(RetainBlockNumberFlag.Name)

	if uint64(uint32Max) < flagValue {
		return fmt.Errorf("retain blocks value overflows uint32 boundaries, must be less than or equal to: %d", uint32Max)
	}

	retainBlocks := uint32(flagValue)

	pruner, err := state.NewOfflinePruner(inputDBPath, prunedDBPath, bloomSize, retainBlocks)
	if err != nil {
		return err
	}

	logger.Info("Offline pruner initialised")

	err = pruner.SetBloomFilter()
	if err != nil {
		return fmt.Errorf("failed to set keys into bloom filter: %w", err)
	}

	err = pruner.Prune()
	if err != nil {
		return fmt.Errorf("failed to prune: %w", err)
	}

	return nil
}
