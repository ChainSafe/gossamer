// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	dotsync "github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "dot"))

// Node is a container for all the components of a node.
type Node struct {
	Name            string
	ServiceRegistry ServiceRegisterer // registry of all node services
	wg              sync.WaitGroup
	started         chan struct{}
	metricsServer   *metrics.Server
}

type nodeBuilderIface interface {
	isNodeInitialised(basepath string) (bool, error)
	initNode(config *cfg.Config) error
	createStateService(config *cfg.Config) (*state.Service, error)
	createNetworkService(config *cfg.Config, stateSrvc *state.Service, telemetryMailer Telemetry) (*network.Service,
		error)
	createRuntimeStorage(st *state.Service) (*runtime.NodeStorage, error)
	loadRuntime(config *cfg.Config, ns *runtime.NodeStorage, stateSrvc *state.Service, ks *keystore.GlobalKeystore,
		net *network.Service) error
	createBlockVerifier(st *state.Service) *babe.VerificationManager
	createDigestHandler(st *state.Service) (*digest.Handler, error)
	createCoreService(config *cfg.Config, ks *keystore.GlobalKeystore, st *state.Service, net *network.Service,
		dh *digest.Handler) (*core.Service, error)
	createGRANDPAService(config *cfg.Config, st *state.Service, ks KeyStore,
		net *network.Service, telemetryMailer Telemetry) (*grandpa.Service, error)
	createParachainHostService(net *network.Service, forkID string, genesishHash common.Hash) (*parachain.Service, error)
	newSyncService(config *cfg.Config, st *state.Service, finalityGadget BlockJustificationVerifier,
		verifier *babe.VerificationManager, cs *core.Service, net *network.Service,
		telemetryMailer Telemetry) (*dotsync.Service, error)
	createBABEService(config *cfg.Config, st *state.Service, ks KeyStore, cs *core.Service,
		telemetryMailer Telemetry) (service *babe.Service, err error)
	createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error)
	createRPCService(params rpcServiceSettings) (*rpc.HTTPServer, error)
}

var _ nodeBuilderIface = (*nodeBuilder)(nil)

type nodeBuilder struct{}

// IsNodeInitialised returns true if, within the configured data directory for the
// node, the state database has been created and the genesis data can been loaded
func IsNodeInitialised(basepath string) (bool, error) {
	nodeInstance := nodeBuilder{}
	return nodeInstance.isNodeInitialised(basepath)
}

// isNodeInitialised returns nil if the node is successfully initialised
// and an error otherwise.
func (nodeBuilder) isNodeInitialised(basepath string) (bool, error) {
	// check if key registry exists
	nodeDatabaseDir := filepath.Join(basepath, database.DefaultDatabaseDir)

	_, err := os.Stat(nodeDatabaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	entries, err := os.ReadDir(nodeDatabaseDir)
	if err != nil {
		return false, fmt.Errorf("failed to read dir %s: %w", nodeDatabaseDir, err)
	}

	if len(entries) == 0 {
		return false, nil
	}

	db, err := database.LoadDatabase(basepath, false)
	if err != nil {
		return false, fmt.Errorf("cannot setup database: %w", err)
	}

	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			logger.Errorf("failed to close database: %s", closeErr)
		}
	}()

	_, err = state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		return false, fmt.Errorf("cannot load genesis data in base state: %w", err)
	}

	return true, nil
}

// InitNode initialise the node with the given Config
func InitNode(config *cfg.Config) error {
	nodeInstance := nodeBuilder{}
	return nodeInstance.initNode(config)
}

// InitNode initialises a new dot node from the provided dot node configuration
// and JSON formatted genesis file.
func (nodeBuilder) initNode(config *cfg.Config) error {
	globalLogLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}
	logger.Patch(log.SetLevel(globalLogLevel))
	logger.Infof(
		"🕸️ initialising node with name %s, id %s, base path %s and chain-spec %s...",
		config.Name, config.ID, config.BasePath, config.ChainSpec)

	// create genesis from configuration file
	gen, err := genesis.NewGenesisFromJSONRaw(config.ChainSpec)
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
	t, err := runtime.NewTrieFromGenesis(*gen)
	if err != nil {
		return fmt.Errorf("failed to create trie from genesis: %w", err)
	}

	// create genesis block from trie
	header, err := t.GenesisBlock()
	if err != nil {
		return fmt.Errorf("failed to create genesis block from trie: %w", err)
	}

	telemetryMailer, err := setupTelemetry(config, nil)
	if err != nil {
		return fmt.Errorf("cannot setup telemetry mailer: %w", err)
	}

	stateLogLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		return fmt.Errorf("cannot parse log level: %w", err)
	}

	stateConfig := state.Config{
		Path:     config.BasePath,
		LogLevel: stateLogLevel,
		PrunerCfg: pruner.Config{
			Mode:           config.Pruning,
			RetainedBlocks: config.RetainBlocks,
		},
		Telemetry: telemetryMailer,
		Metrics:   metrics.NewIntervalConfig(config.PrometheusExternal),
	}

	// create new state service
	stateSrvc := state.NewService(stateConfig)

	// initialise state service with genesis data, block, and trie
	err = stateSrvc.Initialise(gen, &header, &t)
	if err != nil {
		return fmt.Errorf("failed to initialise state service: %s", err)
	}

	err = storeGlobalNodeName(config.Name, config.BasePath)
	if err != nil {
		return fmt.Errorf("failed to store global node name: %s", err)
	}

	logger.Infof(
		"node initialised with name %s, id %s, base path %s, chain-spec %s, block %v and genesis hash %s",
		config.Name, config.ID, config.BasePath, config.ChainSpec, header.Number, header.Hash())

	return nil
}

// LoadGlobalNodeName returns the stored global node name from database
func LoadGlobalNodeName(basepath string) (nodename string, err error) {
	// initialise database using data directory
	db, err := database.LoadDatabase(basepath, false)
	if err != nil {
		return "", err
	}

	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			logger.Errorf("failed to close database: %s", closeErr)
			return
		}
	}()

	basestate := state.NewBaseState(db)
	nodename, err = basestate.LoadNodeGlobalName()
	if err != nil {
		logger.Warnf("failed to load global node name from base path %s: %s", basepath, err)
	}
	return nodename, err
}

// NewNode creates a node based on the given Config and key store.
func NewNode(config *cfg.Config, ks *keystore.GlobalKeystore) (*Node, error) {
	serviceRegistryLogger := logger.New(log.AddContext("pkg", "services"))
	return newNode(config, ks, &nodeBuilder{}, services.NewServiceRegistry(serviceRegistryLogger))
}

func newNode(config *cfg.Config,
	ks *keystore.GlobalKeystore,
	builder nodeBuilderIface,
	serviceRegistry ServiceRegisterer) (*Node, error) {
	// set garbage collection percent to 10%
	// can be overwritten by setting the GOGC env variable, which defaults to 100
	prev := debug.SetGCPercent(10)
	if prev != 100 {
		debug.SetGCPercent(prev)
	}

	isInitialised, err := builder.isNodeInitialised(config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("checking if node is initialised: %w", err)
	}

	if !isInitialised {
		err := builder.initNode(config)
		if err != nil {
			return nil, fmt.Errorf("cannot initialise node: %w", err)
		}
	}

	globalLogLevel, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("cannot parse global log level: %w", err)
	}

	logger.Patch(log.SetLevel(globalLogLevel))

	logger.Infof(
		"🕸️ initialising node services with global configuration name %s, id %s and base path %s...",
		config.Name, config.ID, config.BasePath)

	var (
		nodeSrvcs   []service
		networkSrvc *network.Service
	)

	if config.Pprof.Enabled {
		nodeSrvcs = append(nodeSrvcs, createPprofService(*config.Pprof))
	}

	stateSrvc, err := builder.createStateService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create state service: %s", err)
	}

	gd, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, fmt.Errorf("cannot load genesis data: %w", err)
	}

	telemetryMailer, err := setupTelemetry(config, gd)
	if err != nil {
		return nil, fmt.Errorf("cannot setup telemetry mailer: %w", err)
	}

	stateSrvc.Telemetry = telemetryMailer

	err = startStateService(*config.State, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("cannot start state service: %w", err)
	}

	systemInfo := &types.SystemInfo{
		SystemName:    config.System.SystemName,
		SystemVersion: config.System.SystemVersion,
	}

	sysSrvc, err := builder.createSystemService(systemInfo, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create system service: %s", err)
	}

	nodeSrvcs = append(nodeSrvcs, sysSrvc)

	// check if network service is enabled
	if enabled := networkServiceEnabled(config); enabled {
		// create network service and append network service to node services
		networkSrvc, err = builder.createNetworkService(config, stateSrvc, telemetryMailer)
		if err != nil {
			return nil, fmt.Errorf("failed to create network service: %s", err)
		}
		nodeSrvcs = append(nodeSrvcs, networkSrvc)
		startupTime := fmt.Sprint(time.Now().UnixNano())
		genesisHash := stateSrvc.Block.GenesisHash()
		netstate := networkSrvc.NetworkState()

		//sent NewSystemConnectedTM only if networkServiceEnabled
		connectedMsg := telemetry.NewSystemConnected(
			config.Core.GrandpaAuthority,
			sysSrvc.ChainName(),
			&genesisHash,
			sysSrvc.SystemName(),
			config.BaseConfig.Name,
			netstate.PeerID,
			startupTime,
			sysSrvc.SystemVersion())

		telemetryMailer.SendMessage(connectedMsg)
	} else {
		// do not create or append network service if network service is not enabled
		logger.Debugf("network service disabled, role is %d", config.Core.Role)
	}

	// create runtime
	ns, err := builder.createRuntimeStorage(stateSrvc)
	if err != nil {
		return nil, err
	}

	err = builder.loadRuntime(config, ns, stateSrvc, ks, networkSrvc)
	if err != nil {
		return nil, err
	}

	ver := builder.createBlockVerifier(stateSrvc)

	dh, err := builder.createDigestHandler(stateSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, dh)

	coreSrvc, err := builder.createCoreService(config, ks, stateSrvc, networkSrvc, dh)
	if err != nil {
		return nil, fmt.Errorf("failed to create core service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, coreSrvc)

	fg, err := builder.createGRANDPAService(config, stateSrvc, ks.Gran, networkSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, fg)

	phs, err := builder.createParachainHostService(networkSrvc, gd.ForkID, stateSrvc.Block.GenesisHash())
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, phs)

	syncer, err := builder.newSyncService(config, stateSrvc, fg, ver, coreSrvc, networkSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}

	if networkSrvc != nil {
		networkSrvc.SetSyncer(syncer)
		networkSrvc.SetTransactionHandler(coreSrvc)
	}
	nodeSrvcs = append(nodeSrvcs, syncer)

	bp, err := builder.createBABEService(config, stateSrvc, ks.Babe, coreSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, bp)

	// check if rpc service is enabled
	if enabled := config.RPC.IsRPCEnabled() || config.RPC.IsWSEnabled(); enabled {
		var rpcSrvc *rpc.HTTPServer
		cRPCParams := rpcServiceSettings{
			config:        config,
			nodeStorage:   ns,
			state:         stateSrvc,
			core:          coreSrvc,
			network:       networkSrvc,
			blockProducer: bp,
			system:        sysSrvc,
			blockFinality: fg,
			syncer:        syncer,
		}
		rpcSrvc, err = builder.createRPCService(cRPCParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create rpc service: %s", err)
		}
		nodeSrvcs = append(nodeSrvcs, rpcSrvc)
	} else {
		logger.Debug("rpc service disabled by default")
	}

	// close state service last
	nodeSrvcs = append(nodeSrvcs, stateSrvc)

	node := &Node{
		Name:            config.Name,
		ServiceRegistry: serviceRegistry,
		started:         make(chan struct{}),
	}

	for _, srvc := range nodeSrvcs {
		node.ServiceRegistry.RegisterService(srvc)
	}

	if config.PrometheusExternal {
		address := fmt.Sprintf(":%d", config.PrometheusPort)
		node.metricsServer = metrics.NewServer(address)
		err := node.metricsServer.Start(address)
		if err != nil {
			return nil, fmt.Errorf("cannot start metrics server: %w", err)
		}
	}

	return node, nil
}

func setupTelemetry(config *cfg.Config, genesisData *genesis.Data) (mailer Telemetry, err error) {
	if config.NoTelemetry {
		return telemetry.NewNoopMailer(), nil
	}

	var telemetryEndpoints []*genesis.TelemetryEndpoint
	if len(config.TelemetryURLs) == 0 && genesisData != nil {
		telemetryEndpoints = append(telemetryEndpoints, genesisData.TelemetryEndpoints...)
	} else {
		telemetryURLs := config.TelemetryURLs
		for i := range telemetryURLs {
			telemetryEndpoints = append(telemetryEndpoints, &telemetryURLs[i])
		}
	}

	telemetryLogger := log.NewFromGlobal(log.AddContext("pkg", "telemetry"))
	return telemetry.BootstrapMailer(context.TODO(),
		telemetryEndpoints, telemetryLogger)
}

// stores the global node name to reuse
func storeGlobalNodeName(name, basepath string) (err error) {
	db, err := database.LoadDatabase(basepath, false)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			logger.Errorf("failed to close database: %s", closeErr)
			return
		}
	}()

	basestate := state.NewBaseState(db)
	err = basestate.StoreNodeGlobalName(name)
	if err != nil {
		logger.Warnf("failed to store global node name at base path %s: %s", basepath, err)
		return err
	}

	return nil
}

// Start starts all dot node services
func (n *Node) Start() error {
	logger.Info("🕸️ starting node services...")

	// start all dot node services
	n.ServiceRegistry.StartAll()

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
	// stop all node services
	n.ServiceRegistry.StopAll()
	n.wg.Done()
	if n.metricsServer != nil {
		err := n.metricsServer.Stop()
		if err != nil {
			log.Errorf("cannot stop metrics server: %s", err)
		}
	}
}

func (nodeBuilder) loadRuntime(config *cfg.Config, ns *runtime.NodeStorage,
	stateSrvc *state.Service, ks *keystore.GlobalKeystore,
	net *network.Service) error {
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

		rt, err := createRuntime(config, *ns, stateSrvc, ks, net, code)
		if err != nil {
			return err
		}

		runtimeCode[codeHash.String()] = rt
	}

	return nil
}
