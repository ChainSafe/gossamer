// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ChainSafe/gossamer/chain/dev"
	"github.com/ChainSafe/gossamer/chain/gssmr"
	"github.com/ChainSafe/gossamer/dot"
	ctoml "github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime/life"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/urfave/cli"
)

var (
	// DefaultCfg is the default configuration for the node.
	DefaultCfg                = dot.GssmrConfig
	defaultGssmrConfigPath    = "./chain/gssmr/config.toml"
	defaultKusamaConfigPath   = "./chain/kusama/config.toml"
	defaultPolkadotConfigPath = "./chain/polkadot/config.toml"
	defaultDevConfigPath      = "./chain/dev/config.toml"

	gossamerName = "gssmr"
	kusamaName   = "kusama"
	polkadotName = "polkadot"
	devName      = "dev"
)

// loadConfigFile loads a default config file if --chain is specified, a specific
// config if --config is specified, or the default gossamer config otherwise.
func loadConfigFile(ctx *cli.Context, cfg *ctoml.Config) (err error) {
	// check --config flag and load toml configuration from config.toml
	if cfgPath := ctx.GlobalString(ConfigFlag.Name); cfgPath != "" {
		logger.Info("loading toml configuration from " + cfgPath + "...")
		if cfg == nil {
			cfg = &ctoml.Config{} // if configuration not set, create empty configuration
		} else {
			logger.Warn(
				"overwriting default configuration with id " + cfg.Global.ID +
					" with toml configuration values from " + cfgPath)
		}
		err = loadConfig(cfg, cfgPath) // load toml values into configuration
	} else {
		err = loadConfig(cfg, defaultGssmrConfigPath)
	}

	return err
}

func setupConfigFromChain(ctx *cli.Context) (*ctoml.Config, *dot.Config, error) {
	tomlCfg := &ctoml.Config{}
	cfg := DefaultCfg()

	err := loadConfigFile(ctx, tomlCfg)
	if err != nil {
		logger.Errorf("failed to load toml configuration: %s", err)
		return nil, nil, err
	}

	// check --chain flag and load configuration from defaults.go
	if id := ctx.GlobalString(ChainFlag.Name); id != "" {
		switch id {
		case gossamerName:
			logger.Info("loading toml configuration from " + defaultGssmrConfigPath + "...")
			tomlCfg = &ctoml.Config{}
			err = loadConfig(tomlCfg, defaultGssmrConfigPath)
		case kusamaName:
			logger.Info("loading toml configuration from " + defaultKusamaConfigPath + "...")
			tomlCfg = &ctoml.Config{}
			cfg = dot.KusamaConfig()
			err = loadConfig(tomlCfg, defaultKusamaConfigPath)
		case polkadotName:
			logger.Info("loading toml configuration from " + defaultPolkadotConfigPath + "...")
			tomlCfg = &ctoml.Config{}
			cfg = dot.PolkadotConfig()
			err = loadConfig(tomlCfg, defaultPolkadotConfigPath)
		case devName:
			logger.Info("loading toml configuration from " + defaultDevConfigPath + "...")
			tomlCfg = &ctoml.Config{}
			cfg = dot.DevConfig()
			err = loadConfig(tomlCfg, defaultDevConfigPath)
		default:
			return nil, nil, fmt.Errorf("unknown chain id provided: %s", id)
		}
	}

	if err != nil {
		logger.Errorf("failed to set chain configuration: %s", err)
		return nil, nil, err
	}

	return tomlCfg, cfg, nil
}

// createDotConfig creates a new dot configuration from the provided flag values
func createDotConfig(ctx *cli.Context) (*dot.Config, error) {
	tomlCfg, cfg, err := setupConfigFromChain(ctx)
	if err != nil {
		logger.Errorf("failed to set chain configuration: %s", err)
		return nil, err
	}

	// set log config
	err = setLogConfig(ctx, tomlCfg, &cfg.Global, &cfg.Log)
	if err != nil {
		logger.Errorf("failed to set log configuration: %s", err)
		return nil, err
	}

	logger.Infof("loaded package log configuration: %v", cfg.Log)

	// set global configuration values
	if err := setDotGlobalConfig(ctx, tomlCfg, &cfg.Global); err != nil {
		logger.Errorf("failed to set global node configuration: %s", err)
		return nil, err
	}

	// set remaining cli configuration values
	setDotInitConfig(ctx, tomlCfg.Init, &cfg.Init)
	setDotAccountConfig(ctx, tomlCfg.Account, &cfg.Account)
	setDotCoreConfig(ctx, tomlCfg.Core, &cfg.Core)
	setDotNetworkConfig(ctx, tomlCfg.Network, &cfg.Network)
	setDotRPCConfig(ctx, tomlCfg.RPC, &cfg.RPC)
	setDotPprofConfig(ctx, tomlCfg.Pprof, &cfg.Pprof)

	if rewind := ctx.GlobalInt(RewindFlag.Name); rewind != 0 {
		cfg.State.Rewind = rewind
	}

	// set system info
	setSystemInfoConfig(ctx, cfg)

	return cfg, nil
}

// createInitConfig creates the configuration required to initialise a dot node
func createInitConfig(ctx *cli.Context) (*dot.Config, error) {
	tomlCfg, cfg, err := setupConfigFromChain(ctx)
	if err != nil {
		logger.Errorf("failed to set chain configuration: %s", err)
		return nil, err
	}

	// set global configuration values
	err = setDotGlobalConfig(ctx, tomlCfg, &cfg.Global)
	if err != nil {
		logger.Errorf("failed to set global node configuration: %s", err)
		return nil, err
	}

	if !cfg.Global.Pruning.IsValid() {
		return nil, fmt.Errorf("--%s must be either %s or %s", PruningFlag.Name, pruner.Full, pruner.Archive)
	}

	if cfg.Global.RetainBlocks < dev.DefaultRetainBlocks {
		return nil, fmt.Errorf("--%s cannot be less than %d", RetainBlockNumberFlag.Name, dev.DefaultRetainBlocks)
	}

	// set log config
	err = setLogConfig(ctx, tomlCfg, &cfg.Global, &cfg.Log)
	if err != nil {
		logger.Errorf("failed to set log configuration: %s", err)
		return nil, err
	}

	// set init configuration values
	setDotInitConfig(ctx, tomlCfg.Init, &cfg.Init)

	// set system info
	setSystemInfoConfig(ctx, cfg)

	// set core config since BABE values are needed
	setDotCoreConfig(ctx, tomlCfg.Core, &cfg.Core)

	// ensure configuration values match genesis and overwrite with genesis
	updateDotConfigFromGenesisJSONRaw(*tomlCfg, cfg)

	// set network config here otherwise it's values will be overwritten when starting the node.
	// See /cmd/gossamer/main.go L192.
	setDotNetworkConfig(ctx, tomlCfg.Network, &cfg.Network)

	return cfg, nil
}

func createImportStateConfig(ctx *cli.Context) (*dot.Config, error) {
	tomlCfg, cfg, err := setupConfigFromChain(ctx)
	if err != nil {
		logger.Errorf("failed to set chain configuration: %s", err)
		return nil, err
	}

	// set global configuration values
	if err := setDotGlobalConfig(ctx, tomlCfg, &cfg.Global); err != nil {
		logger.Errorf("failed to set global node configuration: %s", err)
		return nil, err
	}

	return cfg, nil
}

func createBuildSpecConfig(ctx *cli.Context) (*dot.Config, error) {
	var tomlCfg *ctoml.Config
	cfg := &dot.Config{}
	err := loadConfigFile(ctx, tomlCfg)
	if err != nil {
		logger.Errorf("failed to load toml configuration: %s", err)
		return nil, err
	}

	// set global configuration values
	if err := setDotGlobalConfig(ctx, tomlCfg, &cfg.Global); err != nil {
		logger.Errorf("failed to set global node configuration: %s", err)
		return nil, err
	}

	return cfg, nil
}

// createExportConfig creates a new dot configuration from the provided flag values
func createExportConfig(ctx *cli.Context) (*dot.Config, error) {
	cfg := DefaultCfg() // start with default configuration
	tomlCfg := &ctoml.Config{}

	err := loadConfigFile(ctx, tomlCfg)
	if err != nil {
		logger.Errorf("failed to load toml configuration: %s", err)
		return nil, err
	}

	// ensure configuration values match genesis and overwrite with genesis
	updateDotConfigFromGenesisJSONRaw(*tomlCfg, cfg)

	// set global configuration values
	err = setDotGlobalConfig(ctx, tomlCfg, &cfg.Global)
	if err != nil {
		logger.Errorf("failed to set global node configuration: %s", err)
		return nil, err
	}

	// set log config
	err = setLogConfig(ctx, &ctoml.Config{}, &cfg.Global, &cfg.Log)
	if err != nil {
		logger.Errorf("failed to set log configuration: %s", err)
		return nil, err
	}

	// set init configuration values
	setDotInitConfig(ctx, tomlCfg.Init, &cfg.Init)

	// set cli configuration values
	setDotAccountConfig(ctx, tomlCfg.Account, &cfg.Account)
	setDotCoreConfig(ctx, tomlCfg.Core, &cfg.Core)
	setDotNetworkConfig(ctx, tomlCfg.Network, &cfg.Network)
	setDotRPCConfig(ctx, tomlCfg.RPC, &cfg.RPC)

	// set system info
	setSystemInfoConfig(ctx, cfg)

	return cfg, nil
}

type stringKVStore interface {
	String(key string) (value string)
}

// getLogLevel obtains the log level in the following order:
// 1. Try to obtain it from the flag value corresponding to flagName.
// 2. Try to obtain it from the TOML value given, if step 1. failed.
// 3. Return the default value given if both previous steps failed.
// For steps 1 and 2, it tries to parse the level as an integer to convert it
// to a level, and also tries to parse it as a string.
func getLogLevel(flagsKVStore stringKVStore, flagName, tomlValue string, defaultLevel log.Level) (
	level log.Level, err error) {
	if flagValue := flagsKVStore.String(flagName); flagValue != "" {
		return parseLogLevelString(flagValue)
	}

	if tomlValue == "" {
		return defaultLevel, nil
	}

	return parseLogLevelString(tomlValue)
}

var ErrLogLevelIntegerOutOfRange = errors.New("log level integer can only be between 0 and 5 included")

func parseLogLevelString(logLevelString string) (logLevel log.Level, err error) {
	levelInt, err := strconv.Atoi(logLevelString)
	if err == nil { // level given as an integer
		if levelInt < 0 || levelInt > 5 {
			return 0, fmt.Errorf("%w: log level given: %d", ErrLogLevelIntegerOutOfRange, levelInt)
		}
		logLevel = log.Level(levelInt)
		return logLevel, nil
	}

	logLevel, err = log.ParseLevel(logLevelString)
	if err != nil {
		return 0, fmt.Errorf("cannot parse log level string: %w", err)
	}

	return logLevel, nil
}

func setLogConfig(flagsKVStore stringKVStore, tomlConfig *ctoml.Config, globalCfg *dot.GlobalConfig, logCfg *dot.LogConfig) (err error) {
	if tomlConfig == nil {
		tomlConfig = new(ctoml.Config)
	}

	globalCfg.LogLvl, err = getLogLevel(flagsKVStore, LogFlag.Name, tomlConfig.Global.LogLvl, gssmr.DefaultLvl)
	if err != nil {
		return fmt.Errorf("cannot get global log level: %w", err)
	}
	tomlConfig.Global.LogLvl = globalCfg.LogLvl.String()

	levelsData := []struct {
		name      string
		flagName  string
		tomlValue string
		levelPtr  *log.Level // pointer to value to modify
	}{
		{
			name:      "core",
			flagName:  LogCoreLevelFlag.Name,
			tomlValue: tomlConfig.Log.CoreLvl,
			levelPtr:  &logCfg.CoreLvl,
		},
		{
			name:      "sync",
			flagName:  LogSyncLevelFlag.Name,
			tomlValue: tomlConfig.Log.SyncLvl,
			levelPtr:  &logCfg.SyncLvl,
		},
		{
			name:      "network",
			flagName:  LogNetworkLevelFlag.Name,
			tomlValue: tomlConfig.Log.NetworkLvl,
			levelPtr:  &logCfg.NetworkLvl,
		},
		{
			name:      "RPC",
			flagName:  LogRPCLevelFlag.Name,
			tomlValue: tomlConfig.Log.RPCLvl,
			levelPtr:  &logCfg.RPCLvl,
		},
		{
			name:      "state",
			flagName:  LogStateLevelFlag.Name,
			tomlValue: tomlConfig.Log.StateLvl,
			levelPtr:  &logCfg.StateLvl,
		},
		{
			name:      "runtime",
			flagName:  LogRuntimeLevelFlag.Name,
			tomlValue: tomlConfig.Log.RuntimeLvl,
			levelPtr:  &logCfg.RuntimeLvl,
		},
		{
			name:      "block producer",
			flagName:  LogBabeLevelFlag.Name,
			tomlValue: tomlConfig.Log.BlockProducerLvl,
			levelPtr:  &logCfg.BlockProducerLvl,
		},
		{
			name:      "finality gadget",
			flagName:  LogGrandpaLevelFlag.Name,
			tomlValue: tomlConfig.Log.FinalityGadgetLvl,
			levelPtr:  &logCfg.FinalityGadgetLvl,
		},
		{
			name:      "sync",
			flagName:  LogSyncLevelFlag.Name,
			tomlValue: tomlConfig.Log.SyncLvl,
			levelPtr:  &logCfg.SyncLvl,
		},
	}

	for _, levelData := range levelsData {
		level, err := getLogLevel(flagsKVStore, levelData.flagName, levelData.tomlValue, globalCfg.LogLvl)
		if err != nil {
			return fmt.Errorf("cannot get %s log level: %w", levelData.name, err)
		}
		*levelData.levelPtr = level
	}

	logger.Debugf("set log configuration: --log %s global %s", flagsKVStore.String(LogFlag.Name), globalCfg.LogLvl)
	return nil
}

// setDotInitConfig sets dot.InitConfig using flag values from the cli context
func setDotInitConfig(ctx *cli.Context, tomlCfg ctoml.InitConfig, cfg *dot.InitConfig) {
	if tomlCfg.Genesis != "" {
		cfg.Genesis = tomlCfg.Genesis
	}

	// check --genesis flag and update init configuration
	if genesis := ctx.String(GenesisFlag.Name); genesis != "" {
		cfg.Genesis = genesis
	}

	logger.Debug("init configuration with genesis " + cfg.Genesis)
}

func setDotGlobalConfig(ctx *cli.Context, tomlConfig *ctoml.Config, cfg *dot.GlobalConfig) error {
	setDotGlobalConfigFromToml(tomlConfig, cfg)
	if err := setDotGlobalConfigFromFlags(ctx, cfg); err != nil {
		return fmt.Errorf("could not set global config from flags: %w", err)
	}

	if err := setDotGlobalConfigName(ctx, tomlConfig, cfg); err != nil {
		return fmt.Errorf("could not set global node name: %w", err)
	}

	logger.Debug("global configuration has name " + cfg.Name +
		", id " + cfg.ID + " and base path " + cfg.BasePath)

	return nil
}

// setDotGlobalConfigFromToml will apply the toml configs to dot global config
func setDotGlobalConfigFromToml(tomlCfg *ctoml.Config, cfg *dot.GlobalConfig) {
	if tomlCfg != nil {
		if tomlCfg.Global.ID != "" {
			cfg.ID = tomlCfg.Global.ID
		}

		if tomlCfg.Global.BasePath != "" {
			cfg.BasePath = tomlCfg.Global.BasePath
		}

		if tomlCfg.Global.LogLvl != "" {
			level, err := parseLogLevelString(tomlCfg.Global.LogLvl)
			if err == nil {
				cfg.LogLvl = level
			}
		}

		cfg.MetricsPort = tomlCfg.Global.MetricsPort

		cfg.RetainBlocks = tomlCfg.Global.RetainBlocks
		cfg.Pruning = pruner.Mode(tomlCfg.Global.Pruning)
	}
}

// setDotGlobalConfigFromFlags sets dot.GlobalConfig using flag values from the cli context
func setDotGlobalConfigFromFlags(ctx *cli.Context, cfg *dot.GlobalConfig) error {
	// check --basepath flag and update node configuration
	if basepath := ctx.GlobalString(BasePathFlag.Name); basepath != "" {
		cfg.BasePath = basepath
	}

	// check if cfg.BasePath his been set, if not set to default
	if cfg.BasePath == "" {
		cfg.BasePath = dot.GssmrConfig().Global.BasePath
	}

	// check --log flag
	logLevel, err := parseLogLevelString(ctx.String(LogFlag.Name))
	if err == nil {
		cfg.LogLvl = logLevel
	}

	cfg.PublishMetrics = ctx.Bool("publish-metrics")

	// check --metrics-port flag and update node configuration
	if metricsPort := ctx.GlobalUint(MetricsPortFlag.Name); metricsPort != 0 {
		cfg.MetricsPort = uint32(metricsPort)
	}

	cfg.RetainBlocks = ctx.Int64(RetainBlockNumberFlag.Name)
	cfg.Pruning = pruner.Mode(ctx.String(PruningFlag.Name))
	cfg.NoTelemetry = ctx.Bool("no-telemetry")

	var telemetryEndpoints []genesis.TelemetryEndpoint
	for _, telemetryURL := range ctx.GlobalStringSlice(TelemetryURLFlag.Name) {
		splits := strings.Split(telemetryURL, " ")
		if len(splits) != 2 {
			return fmt.Errorf("%s must be in the format 'URL VERBOSITY'", TelemetryURLFlag.Name)
		}

		verbosity, err := strconv.Atoi(splits[1])
		if err != nil {
			return fmt.Errorf("could not parse verbosity from %s: %w", TelemetryURLFlag.Name, err)
		}

		telemetryEndpoints = append(telemetryEndpoints, genesis.TelemetryEndpoint{
			Endpoint:  splits[0],
			Verbosity: verbosity,
		})
	}

	cfg.TelemetryURLs = telemetryEndpoints

	return nil
}

func setDotGlobalConfigName(ctx *cli.Context, tomlCfg *ctoml.Config, cfg *dot.GlobalConfig) error {
	globalBasePath := utils.ExpandDir(cfg.BasePath)
	initialised := dot.NodeInitialized(globalBasePath)

	// consider the --name flag as higher priority
	if ctx.GlobalString(NameFlag.Name) != "" {
		cfg.Name = ctx.GlobalString(NameFlag.Name)
		return nil
	}

	// consider the name on config as a second priority
	if tomlCfg.Global.Name != "" {
		cfg.Name = tomlCfg.Global.Name
		return nil
	}

	// if node was previously initialised and is not the init command
	if initialised && ctx.Command.Name != initCommandName {
		var err error
		if cfg.Name, err = dot.LoadGlobalNodeName(globalBasePath); err != nil {
			return err
		}

		if cfg.Name != "" {
			logger.Debug("load global node name \"" + cfg.Name + "\" from database")
			return nil
		}
	}

	cfg.Name = dot.RandomNodeName()
	return nil
}

// setDotAccountConfig sets dot.AccountConfig using flag values from the cli context
func setDotAccountConfig(ctx *cli.Context, tomlCfg ctoml.AccountConfig, cfg *dot.AccountConfig) {
	if tomlCfg.Key != "" {
		cfg.Key = tomlCfg.Key
	}

	if tomlCfg.Unlock != "" {
		cfg.Unlock = tomlCfg.Unlock
	}

	// check --key flag and update node configuration
	if key := ctx.GlobalString(KeyFlag.Name); key != "" {
		cfg.Key = key
	}

	// check --unlock flag and update node configuration
	if unlock := ctx.GlobalString(UnlockFlag.Name); unlock != "" {
		cfg.Unlock = unlock
	}

	logger.Debug("account configuration has key " + cfg.Key +
		" and unlock " + cfg.Unlock)
}

// setDotCoreConfig sets dot.CoreConfig using flag values from the cli context
func setDotCoreConfig(ctx *cli.Context, tomlCfg ctoml.CoreConfig, cfg *dot.CoreConfig) {
	cfg.Roles = tomlCfg.Roles
	cfg.BabeAuthority = tomlCfg.Roles == types.AuthorityRole
	cfg.GrandpaAuthority = tomlCfg.Roles == types.AuthorityRole
	cfg.GrandpaInterval = time.Second * time.Duration(tomlCfg.GrandpaInterval)

	cfg.BABELead = tomlCfg.BABELead
	if ctx.IsSet(BABELeadFlag.Name) {
		cfg.BABELead = ctx.GlobalBool(BABELeadFlag.Name)
	}

	// check --roles flag and update node configuration
	if roles := ctx.GlobalString(RolesFlag.Name); roles != "" {
		// convert string to byte
		n, err := strconv.Atoi(roles)
		b := byte(n)
		if err != nil {
			logger.Errorf("failed to convert Roles to byte: %s", err)
		} else if b == types.AuthorityRole {
			// if roles byte is 4, act as an authority (see Table D.2)
			logger.Debug("authority enabled (roles=4)")
			cfg.Roles = b
		} else if b > types.AuthorityRole {
			// if roles byte is greater than 4, invalid roles byte (see Table D.2)
			logger.Errorf("invalid roles option provided, authority disabled (roles=%d)", b)
		} else {
			// if roles byte is less than 4, do not act as an authority (see Table D.2)
			logger.Debugf("authority disabled (roles=%d)", b)
			cfg.Roles = b
		}
	}

	// to turn on BABE but not grandpa, cfg.Roles must be set to 4
	// and cfg.GrandpaAuthority must be set to false
	if cfg.Roles == types.AuthorityRole && !tomlCfg.BabeAuthority {
		cfg.BabeAuthority = false
	}

	if cfg.Roles == types.AuthorityRole && !tomlCfg.GrandpaAuthority {
		cfg.GrandpaAuthority = false
	}

	if cfg.Roles != types.AuthorityRole {
		cfg.BabeAuthority = false
		cfg.GrandpaAuthority = false
	}

	switch tomlCfg.WasmInterpreter {
	case wasmer.Name:
		cfg.WasmInterpreter = wasmer.Name
	case life.Name:
		cfg.WasmInterpreter = life.Name
	case "":
		cfg.WasmInterpreter = gssmr.DefaultWasmInterpreter
	default:
		cfg.WasmInterpreter = gssmr.DefaultWasmInterpreter
		logger.Warn("invalid wasm interpreter set in config, defaulting to " + gssmr.DefaultWasmInterpreter)
	}

	logger.Debugf(
		"core configuration: babe-authority=%t, grandpa-authority=%t wasm-interpreter=%s grandpa-interval=%s",
		cfg.BabeAuthority, cfg.GrandpaAuthority, cfg.WasmInterpreter, cfg.GrandpaInterval)
}

// setDotNetworkConfig sets dot.NetworkConfig using flag values from the cli context
func setDotNetworkConfig(ctx *cli.Context, tomlCfg ctoml.NetworkConfig, cfg *dot.NetworkConfig) {
	cfg.Port = tomlCfg.Port
	cfg.Bootnodes = tomlCfg.Bootnodes
	cfg.ProtocolID = tomlCfg.ProtocolID
	cfg.NoBootstrap = tomlCfg.NoBootstrap
	cfg.NoMDNS = tomlCfg.NoMDNS
	cfg.MinPeers = tomlCfg.MinPeers
	cfg.MaxPeers = tomlCfg.MaxPeers
	cfg.PersistentPeers = tomlCfg.PersistentPeers
	cfg.DiscoveryInterval = time.Second * time.Duration(tomlCfg.DiscoveryInterval)

	// check --port flag and update node configuration
	if port := ctx.GlobalUint(PortFlag.Name); port != 0 {
		cfg.Port = uint16(port)
	}

	// check --bootnodes flag and update node configuration
	if bootnodes := ctx.GlobalString(BootnodesFlag.Name); bootnodes != "" {
		cfg.Bootnodes = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
	}

	// format bootnodes
	if len(cfg.Bootnodes) == 0 {
		cfg.Bootnodes = []string(nil)
	}

	// check --protocol flag and update node configuration
	if protocol := ctx.GlobalString(ProtocolFlag.Name); protocol != "" {
		cfg.ProtocolID = protocol
	}

	// check --nobootstrap flag and update node configuration
	if nobootstrap := ctx.GlobalBool(NoBootstrapFlag.Name); nobootstrap {
		cfg.NoBootstrap = true
	}

	// check --nomdns flag and update node configuration
	if nomdns := ctx.GlobalBool(NoMDNSFlag.Name); nomdns {
		cfg.NoMDNS = true
	}

	// check --pubip flag and update node configuration
	if pubip := ctx.GlobalString(PublicIPFlag.Name); pubip != "" {
		cfg.PublicIP = pubip
	}

	if len(cfg.PersistentPeers) == 0 {
		cfg.PersistentPeers = []string(nil)
	}

	logger.Debugf(
		"network configuration: port=%d bootnodes=%s protocol=%s nobootstrap=%t "+
			"nomdns=%t minpeers=%d maxpeers=%d persistent-peers=%s "+
			"discovery-interval=%s",
		cfg.Port, strings.Join(cfg.Bootnodes, ","), cfg.ProtocolID, cfg.NoBootstrap,
		cfg.NoMDNS, cfg.MinPeers, cfg.MaxPeers, strings.Join(cfg.PersistentPeers, ","),
		cfg.DiscoveryInterval,
	)
}

// setDotRPCConfig sets dot.RPCConfig using flag values from the cli context
func setDotRPCConfig(ctx *cli.Context, tomlCfg ctoml.RPCConfig, cfg *dot.RPCConfig) {
	cfg.Enabled = tomlCfg.Enabled
	cfg.External = tomlCfg.External
	cfg.Unsafe = tomlCfg.Unsafe
	cfg.UnsafeExternal = tomlCfg.UnsafeExternal
	cfg.Port = tomlCfg.Port
	cfg.Host = tomlCfg.Host
	cfg.Modules = tomlCfg.Modules
	cfg.WSPort = tomlCfg.WSPort
	cfg.WS = tomlCfg.WS
	cfg.WSExternal = tomlCfg.WSExternal
	cfg.WSUnsafe = tomlCfg.WSUnsafe
	cfg.WSUnsafeExternal = tomlCfg.WSUnsafeExternal

	// check --rpc flag and update node configuration
	if enabled := ctx.GlobalBool(RPCEnabledFlag.Name); enabled || cfg.Enabled {
		cfg.Enabled = true
	} else if ctx.IsSet(RPCEnabledFlag.Name) && !enabled {
		cfg.Enabled = false
	}

	// check --rpc-external flag and update node configuration
	if external := ctx.GlobalBool(RPCExternalFlag.Name); external {
		cfg.Enabled = true
		cfg.External = true
	} else if ctx.IsSet(RPCExternalFlag.Name) && !external {
		cfg.Enabled = true
		cfg.External = false
	}

	// check --rpcport flag and update node configuration
	if port := ctx.GlobalUint(RPCPortFlag.Name); port != 0 {
		cfg.Port = uint32(port)
	}

	// check --rpchost flag and update node configuration
	if host := ctx.GlobalString(RPCHostFlag.Name); host != "" {
		cfg.Host = host
	}

	// check --rpcmods flag and update node configuration
	if modules := ctx.GlobalString(RPCModulesFlag.Name); modules != "" {
		cfg.Modules = strings.Split(ctx.GlobalString(RPCModulesFlag.Name), ",")
	}

	if wsport := ctx.GlobalUint(WSPortFlag.Name); wsport != 0 {
		cfg.WSPort = uint32(wsport)
	}

	if WS := ctx.GlobalBool(WSFlag.Name); WS || cfg.WS {
		cfg.WS = true
	} else if ctx.IsSet(WSFlag.Name) && !WS {
		cfg.WS = false
	}

	if wsExternal := ctx.GlobalBool(WSExternalFlag.Name); wsExternal {
		cfg.WS = true
		cfg.WSExternal = true
	} else if ctx.IsSet(WSExternalFlag.Name) && !wsExternal {
		cfg.WS = true
		cfg.WSExternal = false
	}

	// format rpc modules
	if len(cfg.Modules) == 0 {
		cfg.Modules = []string(nil)
	}

	logger.Debugf("rpc configuration: %s", cfg)
}

func setSystemInfoConfig(ctx *cli.Context, cfg *dot.Config) {
	// load system information
	if ctx.App != nil {
		cfg.System.SystemName = ctx.App.Name
		cfg.System.SystemVersion = ctx.App.Version
	}
}

// updateDotConfigFromGenesisJSONRaw updates the configuration based on the raw genesis file values
func updateDotConfigFromGenesisJSONRaw(tomlCfg ctoml.Config, cfg *dot.Config) {
	cfg.Account.Key = tomlCfg.Account.Key
	cfg.Account.Unlock = tomlCfg.Account.Unlock
	cfg.Core.Roles = tomlCfg.Core.Roles
	cfg.Core.BabeAuthority = tomlCfg.Core.Roles == types.AuthorityRole
	cfg.Core.GrandpaAuthority = tomlCfg.Core.Roles == types.AuthorityRole

	// use default genesis file if genesis configuration not provided, for example,
	// if we load a toml configuration file without a defined genesis init value or
	// if we pass an empty string as the genesis init value using the --genesis flag
	if cfg.Init.Genesis == "" {
		cfg.Init.Genesis = DefaultCfg().Init.Genesis
	}

	// load Genesis from genesis configuration file
	gen, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
	if err != nil {
		logger.Errorf("failed to load genesis from file: %s", err)
		return // exit
	}

	cfg.Global.ID = gen.ID
	cfg.Network.Bootnodes = gen.Bootnodes
	cfg.Network.ProtocolID = gen.ProtocolID

	if gen.ProtocolID == "" {
		logger.Critical("empty protocol ID in genesis file, please set it!")
	}

	logger.Debugf(
		"configuration after genesis json:" +
			" name=" + cfg.Global.Name +
			" id=" + cfg.Global.ID +
			" bootnodes=" + strings.Join(cfg.Network.Bootnodes, ",") +
			" protocol=" + cfg.Network.ProtocolID,
	)
}

// updateDotConfigFromGenesisData updates the configuration from genesis data of an initialised node
func updateDotConfigFromGenesisData(ctx *cli.Context, cfg *dot.Config) error {
	// initialise database using data directory
	db, err := utils.SetupDatabase(cfg.Global.BasePath, false)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	// load genesis data from initialised node database
	gen, err := state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		return fmt.Errorf("failed to load genesis data: %s", err)
	}

	// check genesis id and use genesis id if --chain flag not set
	if !ctx.GlobalIsSet(ChainFlag.Name) {
		cfg.Global.ID = gen.ID
	}

	// check genesis bootnodes and use genesis --bootnodes if name flag not set
	if !ctx.GlobalIsSet(BootnodesFlag.Name) {
		cfg.Network.Bootnodes = common.BytesToStringArray(gen.Bootnodes)
	}

	// check genesis protocol and use genesis --protocol if name flag not set
	if !ctx.GlobalIsSet(ProtocolFlag.Name) {
		cfg.Network.ProtocolID = gen.ProtocolID
	}

	// close database
	err = db.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %s", err)
	}

	logger.Debugf(
		"configuration after genesis data:" +
			" name=" + cfg.Global.Name +
			" id=" + cfg.Global.ID +
			" bootnodes=" + strings.Join(cfg.Network.Bootnodes, ",") +
			" protocol=" + cfg.Network.ProtocolID,
	)

	return nil
}

func setDotPprofConfig(ctx *cli.Context, tomlCfg ctoml.PprofConfig, cfg *dot.PprofConfig) {
	if !cfg.Enabled {
		// only allow to enable pprof from the TOML configuration.
		// If it is enabled by default, it cannot be disabled.
		cfg.Enabled = tomlCfg.Enabled
	}

	if tomlCfg.ListeningAddress != "" {
		cfg.Settings.ListeningAddress = tomlCfg.ListeningAddress
	}

	if tomlCfg.BlockRate > 0 {
		// block rate must be 0 (disabled) by default, since we
		// cannot disable it here.
		cfg.Settings.BlockProfileRate = tomlCfg.BlockRate
	}

	if tomlCfg.MutexRate > 0 {
		// mutex rate must be 0 (disabled) by default, since we
		// cannot disable it here.
		cfg.Settings.MutexProfileRate = tomlCfg.MutexRate
	}

	// check --pprofserver flag and update node configuration
	if enabled := ctx.GlobalBool(PprofServerFlag.Name); enabled || cfg.Enabled {
		cfg.Enabled = true
	} else if ctx.IsSet(PprofServerFlag.Name) && !enabled {
		cfg.Enabled = false
	}

	// check --pprofaddress flag and update node configuration
	if address := ctx.GlobalString(PprofAddressFlag.Name); address != "" {
		cfg.Settings.ListeningAddress = address
	}

	if rate := ctx.GlobalInt(PprofBlockRateFlag.Name); rate > 0 {
		cfg.Settings.BlockProfileRate = rate
	}

	if rate := ctx.GlobalInt(PprofMutexRateFlag.Name); rate > 0 {
		cfg.Settings.MutexProfileRate = rate
	}

	logger.Debug("pprof configuration: " + cfg.String())
}
