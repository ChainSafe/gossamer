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

package dot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/utils"
)

var logger log.Interface = log.NewFromGlobal(log.AddContext("pkg", "dot"))

// Node is a container for all the components of a node.
type Node struct {
	Name     string
	Services *services.ServiceRegistry // registry of all node services
	StopFunc func()                    // func to call when node stops, currently used for profiling
	wg       sync.WaitGroup
	started  chan struct{}
}

// InitNode initialises a new dot node from the provided dot node configuration
// and JSON formatted genesis file.
func InitNode(cfg *Config) error {
	logger.PatchLevel(cfg.Global.LogLvl)
	logger.Info(fmt.Sprintf(
		"🕸️ initialising node with name %s, id %s, base path %s and genesis %s...",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath, cfg.Init.Genesis))

	// create genesis from configuration file
	gen, err := genesis.NewGenesisFromJSONRaw(cfg.Init.Genesis)
	if err != nil {
		return fmt.Errorf("failed to load genesis from file: %w", err)
	}

	if !gen.IsRaw() {
		// genesis is human-readable, convert to raw
		err = gen.ToRaw()
		if err != nil {
			return fmt.Errorf("failed to convert genesis-spec to raw genesis: %w", err)
		}
	}

	// create trie from genesis
	t, err := genesis.NewTrieFromGenesis(gen)
	if err != nil {
		return fmt.Errorf("failed to create trie from genesis: %w", err)
	}

	// create genesis block from trie
	header, err := genesis.NewGenesisBlockFromTrie(t)
	if err != nil {
		return fmt.Errorf("failed to create genesis block from trie: %w", err)
	}

	config := state.Config{
		Path:     cfg.Global.BasePath,
		LogLevel: cfg.Global.LogLvl,
		PrunerCfg: pruner.Config{
			Mode:           cfg.Global.Pruning,
			RetainedBlocks: cfg.Global.RetainBlocks,
		},
	}

	// create new state service
	stateSrvc := state.NewService(config)

	// initialise state service with genesis data, block, and trie
	err = stateSrvc.Initialise(gen, header, t)
	if err != nil {
		return fmt.Errorf("failed to initialise state service: %s", err)
	}

	err = storeGlobalNodeName(cfg.Global.Name, cfg.Global.BasePath)
	if err != nil {
		return fmt.Errorf("failed to store global node name: %s", err)
	}

	logger.Info(fmt.Sprintf(
		"node initialised with name %s, id %s, base path %s, genesis %s, block %v and genesis hash %s",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath, cfg.Init.Genesis, header.Number, header.Hash()))

	return nil
}

// NodeInitialized returns true if, within the configured data directory for the
// node, the state database has been created and the genesis data has been loaded
func NodeInitialized(basepath string) bool {
	// check if key registry exists
	registry := path.Join(basepath, utils.DefaultDatabaseDir, "KEYREGISTRY")

	_, err := os.Stat(registry)
	if os.IsNotExist(err) {
		logger.Debug("node has not been initialised from base path " + basepath +
			": failed to locate KEYREGISTRY file in data directory")

		return false
	}

	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		logger.Debug(fmt.Sprintf("failed to create database from base path %s: %s", basepath, err))
		return false
	}

	defer func() {
		// close database
		err = db.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to close database: %s", err))
		}
	}()

	// load genesis data from initialised node database
	_, err = state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		logger.Error(fmt.Sprintf("node has not been initialised from base path %s: %s", basepath, err))
		return false
	}

	return true
}

// LoadGlobalNodeName returns the stored global node name from database
func LoadGlobalNodeName(basepath string) (nodename string, err error) {
	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		return "", err
	}

	defer func() {
		err = db.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to close database: %s", err))
			return
		}
	}()

	basestate := state.NewBaseState(db)
	nodename, err = basestate.LoadNodeGlobalName()
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to load global node name from base path %s: %s", basepath, err))
		return "", err
	}

	return nodename, err
}

// NewNode creates a new dot node from a dot node configuration
func NewNode(cfg *Config, ks *keystore.GlobalKeystore, stopFunc func()) (*Node, error) {
	// set garbage collection percent to 10%
	// can be overwritten by setting the GOGC env variable, which defaults to 100
	prev := debug.SetGCPercent(10)
	if prev != 100 {
		debug.SetGCPercent(prev)
	}

	logger.PatchLevel(cfg.Global.LogLvl)

	// if authority node, should have at least 1 key in keystore
	if cfg.Core.Roles == types.AuthorityRole && (ks.Babe.Size() == 0 || ks.Gran.Size() == 0) {
		return nil, ErrNoKeysProvided
	}

	logger.Info(fmt.Sprintf(
		"🕸️ initialising node services with global configuration name %s, id %s and base path %s...",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath))

	var (
		nodeSrvcs   []services.Service
		networkSrvc *network.Service
	)

	stateSrvc, err := createStateService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create state service: %s", err)
	}

	// check if network service is enabled
	if enabled := networkServiceEnabled(cfg); enabled {
		// create network service and append network service to node services
		networkSrvc, err = createNetworkService(cfg, stateSrvc)
		if err != nil {
			return nil, fmt.Errorf("failed to create network service: %s", err)
		}
		nodeSrvcs = append(nodeSrvcs, networkSrvc)
	} else {
		// do not create or append network service if network service is not enabled
		logger.Debug(fmt.Sprintf("network service disabled, roles are %d", cfg.Core.Roles))
	}

	// create runtime
	ns, err := createRuntimeStorage(stateSrvc)
	if err != nil {
		return nil, err
	}

	err = loadRuntime(cfg, ns, stateSrvc, ks, networkSrvc)
	if err != nil {
		return nil, err
	}

	ver, err := createBlockVerifier(stateSrvc)
	if err != nil {
		return nil, err
	}

	dh, err := createDigestHandler(stateSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, dh)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, networkSrvc, dh)
	if err != nil {
		return nil, fmt.Errorf("failed to create core service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, coreSrvc)

	fg, err := createGRANDPAService(cfg, stateSrvc, dh, ks.Gran, networkSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, fg)

	syncer, err := newSyncService(cfg, stateSrvc, fg, ver, coreSrvc, networkSrvc)
	if err != nil {
		return nil, err
	}

	if networkSrvc != nil {
		networkSrvc.SetSyncer(syncer)
		networkSrvc.SetTransactionHandler(coreSrvc)
	}
	nodeSrvcs = append(nodeSrvcs, syncer)

	bp, err := createBABEService(cfg, stateSrvc, ks.Babe, coreSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, bp)

	sysSrvc, err := createSystemService(&cfg.System, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create system service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, sysSrvc)

	// check if rpc service is enabled
	if enabled := cfg.RPC.isRPCEnabled() || cfg.RPC.isWSEnabled(); enabled {
		var rpcSrvc *rpc.HTTPServer
		rpcSrvc, err = createRPCService(cfg, ns, stateSrvc, coreSrvc, networkSrvc, bp, sysSrvc, fg)
		if err != nil {
			return nil, fmt.Errorf("failed to create rpc service: %s", err)
		}
		nodeSrvcs = append(nodeSrvcs, rpcSrvc)
	} else {
		logger.Debug("rpc service disabled by default")
	}

	// close state service last
	nodeSrvcs = append(nodeSrvcs, stateSrvc)

	serviceRegistryLogger := logger.New(log.AddContext("pkg", "services"))
	node := &Node{
		Name:     cfg.Global.Name,
		StopFunc: stopFunc,
		Services: services.NewServiceRegistry(serviceRegistryLogger),
		started:  make(chan struct{}),
	}

	for _, srvc := range nodeSrvcs {
		node.Services.RegisterService(srvc)
	}

	if cfg.Global.PublishMetrics {
		c := metrics.NewCollector(context.Background())
		c.AddGauge(fg)
		c.AddGauge(stateSrvc)
		c.AddGauge(networkSrvc)

		go c.Start()

		address := fmt.Sprintf("%s:%d", cfg.RPC.Host, cfg.Global.MetricsPort)
		logger.Info("Enabling stand-alone metrics HTTP endpoint at address " + address)
		metrics.PublishMetrics(address)
	}

	gd, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	telemetry.GetInstance().Initialise(!cfg.Global.NoTelemetry)

	var telemetryEndpoints []*genesis.TelemetryEndpoint
	if len(cfg.Global.TelemetryURLs) == 0 {
		telemetryEndpoints = append(telemetryEndpoints, gd.TelemetryEndpoints...)

	} else {
		telemetryURLs := cfg.Global.TelemetryURLs
		for i := range telemetryURLs {
			telemetryEndpoints = append(telemetryEndpoints, &telemetryURLs[i])
		}
	}

	telemetry.GetInstance().AddConnections(telemetryEndpoints)
	genesisHash := stateSrvc.Block.GenesisHash()
	err = telemetry.GetInstance().SendMessage(telemetry.NewSystemConnectedTM(
		cfg.Core.GrandpaAuthority,
		sysSrvc.ChainName(),
		&genesisHash,
		sysSrvc.SystemName(),
		cfg.Global.Name,
		networkSrvc.NetworkState().PeerID,
		strconv.FormatInt(time.Now().UnixNano(), 10),
		sysSrvc.SystemVersion()))
	if err != nil {
		logger.Debug(fmt.Sprintf("problem sending system.connected telemetry message: %s", err))
	}
	return node, nil
}

// stores the global node name to reuse
func storeGlobalNodeName(name, basepath string) (err error) {
	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		return err
	}

	defer func() {
		err = db.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to close database: %s", err))
			return
		}
	}()

	basestate := state.NewBaseState(db)
	err = basestate.StoreNodeGlobalName(name)
	if err != nil {
		logger.Warn(fmt.Sprintf("failed to store global node name at base path %s: %s", basepath, err))
		return err
	}

	return nil
}

// Start starts all dot node services
func (n *Node) Start() error {
	logger.Info("🕸️ starting node services...")

	// start all dot node services
	n.Services.StartAll()

	n.wg.Add(1)
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		logger.Info("signal interrupt, shutting down...")
		n.Stop()
	}()

	close(n.started)
	n.wg.Wait()
	return nil
}

// Stop stops all dot node services
func (n *Node) Stop() {
	if n.StopFunc != nil {
		n.StopFunc()
	}

	// stop all node services
	n.Services.StopAll()
	n.wg.Done()
}

func loadRuntime(cfg *Config, ns *runtime.NodeStorage, stateSrvc *state.Service, ks *keystore.GlobalKeystore, net *network.Service) error {
	blocks := stateSrvc.Block.GetNonFinalisedBlocks()
	runtimeCode := make(map[string]runtime.Instance)
	for i := range blocks {
		hash := &blocks[i]
		code, err := stateSrvc.Storage.GetStorageByBlockHash(hash, []byte(":code"))
		if err != nil {
			return err
		}

		codeHash, err := common.Blake2bHash(code)
		if err != nil {
			return err
		}

		if rt, ok := runtimeCode[codeHash.String()]; ok {
			stateSrvc.Block.StoreRuntime(*hash, rt)
			continue
		}

		rt, err := createRuntime(cfg, *ns, stateSrvc, ks, net, code)
		if err != nil {
			return err
		}

		runtimeCode[codeHash.String()] = rt
	}

	return nil
}
