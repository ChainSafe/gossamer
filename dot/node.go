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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	gssmrmetrics "github.com/ChainSafe/gossamer/dot/metrics"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/utils"
	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
)

var logger = log.New("pkg", "dot")

// Node is a container for all the components of a node.
type Node struct {
	Name     string
	Services *services.ServiceRegistry // registry of all node services
	StopFunc func()                    // func to call when node stops, currently used for profiling
	wg       sync.WaitGroup
}

// InitNode initialises a new dot node from the provided dot node configuration
// and JSON formatted genesis file.
func InitNode(cfg *Config) error {
	setupLogger(cfg)
	logger.Info(
		"🕸️ initialising node...",
		"name", cfg.Global.Name,
		"id", cfg.Global.ID,
		"basepath", cfg.Global.BasePath,
		"genesis", cfg.Init.Genesis,
	)

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

	// create new state service
	stateSrvc := state.NewService(cfg.Global.BasePath, cfg.Global.LogLvl)

	// initialise state service with genesis data, block, and trie
	err = stateSrvc.Initialise(gen, header, t)
	if err != nil {
		return fmt.Errorf("failed to initialise state service: %s", err)
	}

	err = storeGlobalNodeName(cfg.Global.Name, cfg.Global.BasePath)
	if err != nil {
		return fmt.Errorf("failed to store global node name: %s", err)
	}

	logger.Info(
		"node initialised",
		"name", cfg.Global.Name,
		"id", cfg.Global.ID,
		"basepath", cfg.Global.BasePath,
		"genesis", cfg.Init.Genesis,
		"block", header.Number,
		"genesis hash", header.Hash(),
	)

	return nil
}

// NodeInitialized returns true if, within the configured data directory for the
// node, the state database has been created and the genesis data has been loaded
func NodeInitialized(basepath string, expected bool) bool {
	// check if key registry exists
	registry := path.Join(basepath, utils.DefaultDatabaseDir, "KEYREGISTRY")

	_, err := os.Stat(registry)
	if os.IsNotExist(err) {
		if expected {
			logger.Debug(
				"node has not been initialised",
				"basepath", basepath,
				"error", "failed to locate KEYREGISTRY file in data directory",
			)
		}
		return false
	}

	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		logger.Error(
			"failed to create database",
			"basepath", basepath,
			"error", err,
		)
		return false
	}

	defer func() {
		// close database
		err = db.Close()
		if err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	// load genesis data from initialised node database
	_, err = state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		logger.Debug(
			"node has not been initialised",
			"basepath", basepath,
			"error", err,
		)
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
			logger.Error("failed to close database", "error", err)
			return
		}
	}()

	basestate := state.NewBaseState(db)
	nodename, err = basestate.LoadNodeGlobalName()
	if err != nil {
		logger.Warn(
			"failed to load global node name",
			"basepath", basepath,
			"error", err,
		)
		return "", err
	}

	return nodename, err
}

// NewNode creates a new dot node from a dot node configuration
func NewNode(cfg *Config, ks *keystore.GlobalKeystore, stopFunc func()) (*Node, error) {
	// set garbage collection percent to 10%
	// can be overwritten by setting the GOGC env veriable, which defaults to 100
	prev := debug.SetGCPercent(10)
	if prev != 100 {
		debug.SetGCPercent(prev)
	}

	setupLogger(cfg)

	// if authority node, should have at least 1 key in keystore
	if cfg.Core.Roles == types.AuthorityRole && (ks.Babe.Size() == 0 || ks.Gran.Size() == 0) {
		return nil, ErrNoKeysProvided
	}

	// Node Services

	logger.Info(
		"🕸️ initialising node services...",
		"name", cfg.Global.Name,
		"id", cfg.Global.ID,
		"basepath", cfg.Global.BasePath,
	)

	var nodeSrvcs []services.Service

	// State Service

	// create state service and append state service to node services
	stateSrvc, err := createStateService(cfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create state service: %s", err)
	}

	// Network Service
	var networkSrvc *network.Service

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
		logger.Debug("network service disabled", "network", enabled, "roles", cfg.Core.Roles)
	}

	// create runtime
	rt, err := createRuntime(cfg, stateSrvc, ks, networkSrvc)
	if err != nil {
		return nil, err
	}

	ver, err := createBlockVerifier(stateSrvc)
	if err != nil {
		return nil, err
	}

	// create BABE service
	bp, err := createBABEService(cfg, rt, stateSrvc, ks.Babe)
	if err != nil {
		return nil, err
	}

	nodeSrvcs = append(nodeSrvcs, bp)

	dh, err := createDigestHandler(stateSrvc, bp, ver)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, dh)

	// create GRANDPA service
	fg, err := createGRANDPAService(cfg, rt, stateSrvc, dh, ks.Gran, networkSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, fg)

	// Syncer
	syncer, err := createSyncService(cfg, stateSrvc, bp, fg, dh, ver, rt)
	if err != nil {
		return nil, err
	}

	// Core Service

	// create core service and append core service to node services
	coreSrvc, err := createCoreService(cfg, bp, ver, rt, ks, stateSrvc, networkSrvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create core service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, coreSrvc)

	if networkSrvc != nil {
		networkSrvc.SetSyncer(syncer)
		networkSrvc.SetTransactionHandler(coreSrvc)
	}

	// System Service

	// create system service and append to node services
	sysSrvc, err := createSystemService(&cfg.System, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create system service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, sysSrvc)

	// RPC Service

	// check if rpc service is enabled
	if enabled := cfg.RPC.Enabled; enabled {
		// create rpc service and append rpc service to node services
		rpcSrvc := createRPCService(cfg, stateSrvc, coreSrvc, networkSrvc, bp, rt, sysSrvc)
		nodeSrvcs = append(nodeSrvcs, rpcSrvc)
	} else {
		// do not create or append rpc service if rpc service is not enabled
		logger.Debug("rpc service disabled by default", "rpc", enabled)
	}

	// close state service last
	nodeSrvcs = append(nodeSrvcs, stateSrvc)

	node := &Node{
		Name:     cfg.Global.Name,
		StopFunc: stopFunc,
		Services: services.NewServiceRegistry(),
	}

	for _, srvc := range nodeSrvcs {
		node.Services.RegisterService(srvc)
	}

	if cfg.Global.PublishMetrics {
		publishMetrics(cfg)
	}

	gd, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	if cfg.Global.NoTelemetry {
		return node, nil
	}

	telemetry.GetInstance().AddConnections(gd.TelemetryEndpoints)
	data := &telemetry.ConnectionData{
		Authority:     cfg.Core.GrandpaAuthority,
		Chain:         sysSrvc.ChainName(),
		GenesisHash:   stateSrvc.Block.GenesisHash().String(),
		SystemName:    sysSrvc.SystemName(),
		NodeName:      cfg.Global.Name,
		SystemVersion: sysSrvc.SystemVersion(),
		NetworkID:     networkSrvc.NetworkState().PeerID,
		StartTime:     strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	telemetry.GetInstance().SendConnection(data)

	return node, nil
}

func publishMetrics(cfg *Config) {
	address := fmt.Sprintf("%s:%d", cfg.RPC.Host, cfg.Global.MetricsPort)
	log.Info("Enabling stand-alone metrics HTTP endpoint", "address", address)
	setupMetricsServer(address)

	// Start system runtime metrics collection
	go gssmrmetrics.CollectProcessMetrics()
}

// setupMetricsServer starts a dedicated metrics server at the given address.
func setupMetricsServer(address string) {
	m := http.NewServeMux()
	m.Handle("/metrics", prometheus.Handler(metrics.DefaultRegistry))
	log.Info("Starting metrics server", "addr", fmt.Sprintf("http://%s/metrics", address))
	go func() {
		if err := http.ListenAndServe(address, m); err != nil {
			log.Error("Failure in running metrics server", "err", err)
		}
	}()
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
			logger.Error("failed to close database", "error", err)
			return
		}
	}()

	basestate := state.NewBaseState(db)
	err = basestate.StoreNodeGlobalName(name)
	if err != nil {
		logger.Warn(
			"failed to store global node name",
			"basepath", basepath,
			"error", err,
		)
		return err
	}

	return nil
}

// Start starts all dot node services
func (n *Node) Start() error {
	logger.Info("🕸️ starting node services...")

	// start all dot node services
	n.Services.StartAll()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		logger.Info("signal interrupt, shutting down...")
		n.Stop()
		os.Exit(130)
	}()

	n.wg.Add(1)
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
