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
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/ChainSafe/gossamer/state"

	"github.com/ChainSafe/gossamer/cmd/utils"
	"github.com/ChainSafe/gossamer/common"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	"github.com/ChainSafe/gossamer/runtime"
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

// makeNode sets up node; opening badgerDB instance and returning the Dot container
func makeNode(ctx *cli.Context) (*dot.Dot, *cfg.Config, error) {
	defaultConfig, err := getConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	var srvcs []services.Service

	dataDir := expandTilde(defaultConfig.Global.DataDir)

	// Create service, initialize stateDB and blockDB
	stateSrv := state.NewService(dataDir)
	srvcs = append(srvcs, stateSrv)

	err = stateSrv.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot start db service: %s", err)
	}

	// load all static keys from keystore directory
	ks := keystore.NewKeystore()
	// unlock keys, if specified
	if keyindices := ctx.String(utils.UnlockFlag.Name); keyindices != "" {
		err = unlockKeys(ctx, dataDir, ks)
		if err != nil {
			return nil, nil, fmt.Errorf("could not unlock keys: %s", err)
		}
	}

	// Trie, runtime: load most recent state from DB, load runtime code from trie and create runtime executor
	r, err := loadStateAndRuntime(stateSrv.Storage, ks)
	if err != nil {
		return nil, nil, fmt.Errorf("error loading state and runtime: %s", err)
	}

	// load extra genesis data from DB
	gendata, err := stateSrv.Storage.LoadGenesisData()
	if err != nil {
		return nil, nil, err
	}

	log.Info("🕸\t Configuring node...", "dataDir", dataDir, "protocolID", string(gendata.ProtocolID), "BootstrapNodes", defaultConfig.P2p.BootstrapNodes)

	// P2P
	p2pSrvc, p2pMsgSend, p2pMsgRec := createP2PService(defaultConfig, gendata)
	srvcs = append(srvcs, p2pSrvc)

	// Core
	coreConfig := &core.Config{
		Keystore: ks,
		Runtime:  r,
		MsgRec:   p2pMsgSend, // message channel from p2p service to core service
		MsgSend:  p2pMsgRec,  // message channel from core service to p2p service
	}
	coreSrvc := createCoreService(coreConfig)
	srvcs = append(srvcs, coreSrvc)

	// API
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	srvcs = append(srvcs, apiSrvc)

	// RPC
	rpcSrvr := startRpc(ctx, defaultConfig.Rpc, apiSrvc)

	return dot.NewDot(string(gendata.Name), srvcs, rpcSrvr), defaultConfig, nil
}

func loadStateAndRuntime(ss *state.StorageState, ks *keystore.Keystore) (*runtime.Runtime, error) {
	latestState, err := ss.LoadHash()
	if err != nil {
		return nil, fmt.Errorf("cannot load latest state root hash: %s", err)
	}

	err = ss.LoadFromDB(latestState)
	if err != nil {
		return nil, fmt.Errorf("cannot load latest state: %s", err)
	}

	code, err := ss.GetStorage([]byte(":code"))
	if err != nil {
		return nil, fmt.Errorf("error retrieving :code from trie: %s", err)
	}

	return runtime.NewRuntime(code, ss, ks)
}

// getConfig checks for config.toml if --config flag is specified and sets CLI flags
func getConfig(ctx *cli.Context) (*cfg.Config, error) {
	defaultConfig := cfg.DefaultConfig()
	// Load config file.
	if file := ctx.GlobalString(utils.ConfigFileFlag.Name); file != "" {
		configFile := ctx.GlobalString(utils.ConfigFileFlag.Name)
		log.Debug("Loading config file", "configFile", configFile)
		err := loadConfig(configFile, defaultConfig)
		if err != nil {
			log.Warn("err loading toml file", "err", err.Error())
			return defaultConfig, err
		}
	} else {
		log.Debug("Config File is not set")
	}

	// Parse CLI flags
	log.Debug("Parse CLI flags")
	setGlobalConfig(ctx, &defaultConfig.Global)
	setP2pConfig(ctx, &defaultConfig.P2p)
	setRpcConfig(ctx, &defaultConfig.Rpc)
	return defaultConfig, nil
}

// loadConfig loads the contents from config toml and inits Config object
func loadConfig(file string, config *cfg.Config) error {
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	log.Debug("Loading configuration", "path", filepath.Clean(fp))
	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}
	if err = tomlSettings.NewDecoder(f).Decode(&config); err != nil {
		return err
	}
	log.Debug("Loaded configuration", "config", config)
	return nil
}

func setGlobalConfig(ctx *cli.Context, fig *cfg.GlobalConfig) {
	if dir := ctx.GlobalString(utils.DataDirFlag.Name); dir != "" {
		fig.DataDir, _ = filepath.Abs(dir)
	}
	fig.DataDir, _ = filepath.Abs(fig.DataDir)
}

func setP2pConfig(ctx *cli.Context, fig *cfg.P2pCfg) {
	// Bootnodes
	if bnodes := ctx.GlobalString(utils.BootnodesFlag.Name); bnodes != "" {
		fig.BootstrapNodes = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
	}

	if port := ctx.GlobalUint(utils.P2pPortFlag.Name); port != 0 {
		fig.Port = uint32(port)
	}

	// NoBootstrap
	if off := ctx.GlobalBool(utils.NoBootstrapFlag.Name); off {
		fig.NoBootstrap = true
	}

	// NoMdns
	if off := ctx.GlobalBool(utils.NoMdnsFlag.Name); off {
		fig.NoMdns = true
	}
}

// createP2PService creates a p2p service from the command configuration and genesis data
func createP2PService(fig *cfg.Config, gendata *genesis.GenesisData) (*p2p.Service, chan p2p.Message, chan p2p.Message) {
	// Default bootnodes are from genesis
	boostrapNodes := common.BytesToStringArray(gendata.Bootnodes)

	// If bootnodes flag has more than 1 bootnode, overwrite
	if len(fig.P2p.BootstrapNodes) > 0 {
		boostrapNodes = fig.P2p.BootstrapNodes
	}

	// p2p service configuation
	p2pConfig := p2p.Config{
		BootstrapNodes: boostrapNodes,
		Port:           fig.P2p.Port,
		RandSeed:       0,
		NoBootstrap:    fig.P2p.NoBootstrap,
		NoMdns:         fig.P2p.NoMdns,
		DataDir:        fig.Global.DataDir,
		ProtocolID:     string(gendata.ProtocolID),
	}

	p2pMsgRec := make(chan p2p.Message)
	p2pMsgSend := make(chan p2p.Message)

	p2pService, err := p2p.NewService(&p2pConfig, p2pMsgSend, p2pMsgRec)
	if err != nil {
		log.Error("Failed to create new p2p service", "err", err)
	}

	return p2pService, p2pMsgSend, p2pMsgRec
}

// createCoreService creates the core service from the provided core configuration
func createCoreService(coreConfig *core.Config) *core.Service {

	coreBlkRec := make(chan types.Block)

	coreService, err := core.NewService(coreConfig, coreBlkRec)
	if err != nil {
		log.Error("Failed to create new core service", "err", err)
	}

	return coreService
}

func setRpcConfig(ctx *cli.Context, fig *cfg.RpcCfg) {
	// Modules
	if mods := ctx.GlobalString(utils.RpcModuleFlag.Name); mods != "" {
		fig.Modules = strToMods(strings.Split(ctx.GlobalString(utils.RpcModuleFlag.Name), ","))
	}

	// Host
	if host := ctx.GlobalString(utils.RpcHostFlag.Name); host != "" {
		fig.Host = host
	}

	// Port
	if port := ctx.GlobalUint(utils.RpcPortFlag.Name); port != 0 {
		fig.Port = uint32(port)
	}

}

func startRpc(ctx *cli.Context, fig cfg.RpcCfg, apiSrvc *api.Service) *rpc.HttpServer {
	if ctx.GlobalBool(utils.RpcEnabledFlag.Name) {
		return rpc.NewHttpServer(apiSrvc.Api, &json2.Codec{}, fig.Host, fig.Port, fig.Modules)
	}
	return nil
}

// strToMods casts a []strings to []api.Module
func strToMods(strs []string) []api.Module {
	var res []api.Module
	for _, str := range strs {
		res = append(res, api.Module(str))
	}
	return res
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	defaultConfig, err := getConfig(ctx)
	if err != nil {
		return err
	}

	comment := ""

	out, err := toml.Marshal(defaultConfig)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		/* #nosec */
		dump, err = os.OpenFile(filepath.Clean(ctx.Args().Get(0)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		defer func() {
			err = dump.Close()
			if err != nil {
				log.Warn("err closing conn", "err", err.Error())
			}
		}()
	}
	_, err = dump.WriteString(comment)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	_, err = dump.Write(out)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	return nil
}

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

// expandTilde will expand a tilde prefix path to full home path
func expandTilde(targetPath string) string {
	if strings.HasPrefix(targetPath, "~\\") || strings.HasPrefix(targetPath, "~/") {
		if homeDir := home(); homeDir != "" {
			targetPath = homeDir + targetPath[1:]
		}
	}
	return path.Clean(os.ExpandEnv(targetPath))
}

// home will return the HOME ENV DIR OR the current user dir
func home() string {
	if homeDir := os.Getenv("HOME"); homeDir != "" {
		return homeDir
	}
	if usrDir, err := user.Current(); err == nil {
		return usrDir.HomeDir
	}
	return ""
}
