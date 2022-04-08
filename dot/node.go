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

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/state/pruner"
	dotsync "github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/utils"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "dot"))

// Node is a container for all the components of a node.
type Node struct {
	Name            string
	ServiceRegistry services.ServiceRegisterer // registry of all node services
	wg              sync.WaitGroup
	started         chan struct{}
	metricsServer   *metrics.Server
}

//go:generate mockgen -source=node.go -destination=mock_node_builder_test.go -package=$GOPACKAGE

type nodeBuilderIface interface {
	nodeInitialised(string) error
	initNode(config *Config) error
	createStateService(config *Config) (*state.Service, error)
	createNetworkService(cfg *Config, stateSrvc *state.Service, telemetryMailer telemetry.Client) (*network.Service,
		error)
	createRuntimeStorage(st *state.Service) (*runtime.NodeStorage, error)
	loadRuntime(cfg *Config, ns *runtime.NodeStorage, stateSrvc *state.Service, ks *keystore.GlobalKeystore,
		net *network.Service) error
	createBlockVerifier(st *state.Service) (*babe.VerificationManager, error)
	createDigestHandler(lvl log.Level, st *state.Service) (*digest.Handler, error)
	createCoreService(cfg *Config, ks *keystore.GlobalKeystore, st *state.Service, net *network.Service,
		dh *digest.Handler) (*core.Service, error)
	createGRANDPAService(cfg *Config, st *state.Service, dh *digest.Handler, ks keystore.Keystore,
		net *network.Service, telemetryMailer telemetry.Client) (*grandpa.Service, error)
	newSyncService(cfg *Config, st *state.Service, fg dotsync.FinalityGadget, verifier *babe.VerificationManager,
		cs *core.Service, net *network.Service, telemetryMailer telemetry.Client) (*dotsync.Service, error)
	createBABEService(cfg *Config, st *state.Service, ks keystore.Keystore, cs *core.Service,
		telemetryMailer telemetry.Client) (babe.ServiceIFace, error)
	createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error)
	createRPCService(params rpcServiceSettings) (*rpc.HTTPServer, error)
}

var _ nodeBuilderIface = (*nodeBuilder)(nil)

type nodeBuilder struct{}

// NodeInitialized returns true if, within the configured data directory for the
// node, the state database has been created and the genesis data has been loaded
func NodeInitialized(basepath string) bool {
	nodeInstance := nodeBuilder{}
	err := nodeInstance.nodeInitialised(basepath)
	if err != nil {
		logger.Errorf("failed to initialise node from base path %s: %s", basepath, err)
		return false
	}
	return true
}

func (*nodeBuilder) nodeInitialised(basepath string) error {
	// check if key registry exists
	registry := filepath.Join(basepath, utils.DefaultDatabaseDir, "KEYREGISTRY")

	_, err := os.Stat(registry)
	if os.IsNotExist(err) {
		return fmt.Errorf("cannot find key registry in database directory: %w", err)
	}

	db, err := utils.SetupDatabase(basepath, false)
	if err != nil {
		return fmt.Errorf("cannot setup database: %w", err)
	}

	defer func() {
		closeErr := db.Close()
		if err != nil {
			logger.Errorf("failed to close database: %s", closeErr)
		}
	}()

	_, err = state.NewBaseState(db).LoadGenesisData()
	if err != nil {
		return fmt.Errorf("cannot load genesis data in base state: %w", err)
	}

	return nil
}

// InitNode initialise the node with the given Config
func InitNode(cfg *Config) error {
	nodeInstance := nodeBuilder{}
	return nodeInstance.initNode(cfg)
}

// InitNode initialises a new dot node from the provided dot node configuration
// and JSON formatted genesis file.
func (*nodeBuilder) initNode(cfg *Config) error {
	logger.Patch(log.SetLevel(cfg.Global.LogLvl))
	logger.Infof(
		"🕸️ initialising node with name %s, id %s, base path %s and genesis %s...",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath, cfg.Init.Genesis)

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

	telemetryMailer, err := setupTelemetry(cfg, nil)
	if err != nil {
		return fmt.Errorf("cannot setup telemetry mailer: %w", err)
	}

	config := state.Config{
		Path:     cfg.Global.BasePath,
		LogLevel: cfg.Global.LogLvl,
		PrunerCfg: pruner.Config{
			Mode:           cfg.Global.Pruning,
			RetainedBlocks: cfg.Global.RetainBlocks,
		},
		Telemetry: telemetryMailer,
		Metrics:   metrics.NewIntervalConfig(cfg.Global.PublishMetrics),
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

	logger.Infof(
		"node initialised with name %s, id %s, base path %s, genesis %s, block %v and genesis hash %s",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath, cfg.Init.Genesis, header.Number, header.Hash())

	return nil
}

// LoadGlobalNodeName returns the stored global node name from database
func LoadGlobalNodeName(basepath string) (nodename string, err error) {
	// initialise database using data directory
	db, err := utils.SetupDatabase(basepath, false)
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
func NewNode(cfg *Config, ks *keystore.GlobalKeystore) (*Node, error) {
	serviceRegistryLogger := logger.New(log.AddContext("pkg", "services"))
	return newNode(cfg, ks, &nodeBuilder{}, services.NewServiceRegistry(serviceRegistryLogger))
}

func newNode(cfg *Config,
	ks *keystore.GlobalKeystore,
	builder nodeBuilderIface,
	serviceRegistry services.ServiceRegisterer) (*Node, error) {
	// set garbage collection percent to 10%
	// can be overwritten by setting the GOGC env variable, which defaults to 100
	prev := debug.SetGCPercent(10)
	if prev != 100 {
		debug.SetGCPercent(prev)
	}

	if builder.nodeInitialised(cfg.Global.BasePath) != nil {
		err := builder.initNode(cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot initialise node: %w", err)
		}
	}

	logger.Patch(log.SetLevel(cfg.Global.LogLvl))

	logger.Infof(
		"🕸️ initialising node services with global configuration name %s, id %s and base path %s...",
		cfg.Global.Name, cfg.Global.ID, cfg.Global.BasePath)

	var (
		nodeSrvcs   []services.Service
		networkSrvc *network.Service
	)

	if cfg.Pprof.Enabled {
		nodeSrvcs = append(nodeSrvcs, createPprofService(cfg.Pprof.Settings))
	}

	stateSrvc, err := builder.createStateService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create state service: %s", err)
	}

	gd, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, fmt.Errorf("cannot load genesis data: %w", err)
	}

	telemetryMailer, err := setupTelemetry(cfg, gd)
	if err != nil {
		return nil, fmt.Errorf("cannot setup telemetry mailer: %w", err)
	}

	stateSrvc.Telemetry = telemetryMailer

	err = startStateService(cfg, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("cannot start state service: %w", err)
	}

	sysSrvc, err := builder.createSystemService(&cfg.System, stateSrvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create system service: %s", err)
	}

	nodeSrvcs = append(nodeSrvcs, sysSrvc)

	// check if network service is enabled
	if enabled := networkServiceEnabled(cfg); enabled {
		// create network service and append network service to node services
		networkSrvc, err = builder.createNetworkService(cfg, stateSrvc, telemetryMailer)
		if err != nil {
			return nil, fmt.Errorf("failed to create network service: %s", err)
		}
		nodeSrvcs = append(nodeSrvcs, networkSrvc)
		startupTime := fmt.Sprint(time.Now().UnixNano())
		genesisHash := stateSrvc.Block.GenesisHash()
		netstate := networkSrvc.NetworkState()

		//sent NewSystemConnectedTM only if networkServiceEnabled
		connectedMsg := telemetry.NewSystemConnected(
			cfg.Core.GrandpaAuthority,
			sysSrvc.ChainName(),
			&genesisHash,
			sysSrvc.SystemName(),
			cfg.Global.Name,
			netstate.PeerID,
			startupTime,
			sysSrvc.SystemVersion())

		telemetryMailer.SendMessage(connectedMsg)
	} else {
		// do not create or append network service if network service is not enabled
		logger.Debugf("network service disabled, roles are %d", cfg.Core.Roles)
	}

	// create runtime
	ns, err := builder.createRuntimeStorage(stateSrvc)
	if err != nil {
		return nil, err
	}

	err = builder.loadRuntime(cfg, ns, stateSrvc, ks, networkSrvc)
	if err != nil {
		return nil, err
	}

	ver, err := builder.createBlockVerifier(stateSrvc)
	if err != nil {
		return nil, err
	}

	dh, err := builder.createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, dh)

	coreSrvc, err := builder.createCoreService(cfg, ks, stateSrvc, networkSrvc, dh)
	if err != nil {
		return nil, fmt.Errorf("failed to create core service: %s", err)
	}
	nodeSrvcs = append(nodeSrvcs, coreSrvc)

	fg, err := builder.createGRANDPAService(cfg, stateSrvc, dh, ks.Gran, networkSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, fg)

	syncer, err := builder.newSyncService(cfg, stateSrvc, fg, ver, coreSrvc, networkSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}

	if networkSrvc != nil {
		networkSrvc.SetSyncer(syncer)
		networkSrvc.SetTransactionHandler(coreSrvc)
	}
	nodeSrvcs = append(nodeSrvcs, syncer)

	bp, err := builder.createBABEService(cfg, stateSrvc, ks.Babe, coreSrvc, telemetryMailer)
	if err != nil {
		return nil, err
	}
	nodeSrvcs = append(nodeSrvcs, bp)

	// check if rpc service is enabled
	if enabled := cfg.RPC.isRPCEnabled() || cfg.RPC.isWSEnabled(); enabled {
		var rpcSrvc *rpc.HTTPServer
		cRPCParams := rpcServiceSettings{
			config:        cfg,
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
		Name:            cfg.Global.Name,
		ServiceRegistry: serviceRegistry,
		started:         make(chan struct{}),
	}

	for _, srvc := range nodeSrvcs {
		node.ServiceRegistry.RegisterService(srvc)
	}

	if cfg.Global.PublishMetrics {
		node.metricsServer = metrics.NewServer(cfg.Global.MetricsAddress)
		err := node.metricsServer.Start(cfg.Global.MetricsAddress)
		if err != nil {
			return nil, fmt.Errorf("cannot start metrics server: %w", err)
		}
	}

	return node, nil
}

func setupTelemetry(cfg *Config, genesisData *genesis.Data) (mailer *telemetry.Mailer, err error) {
	var telemetryEndpoints []*genesis.TelemetryEndpoint
	if len(cfg.Global.TelemetryURLs) == 0 && genesisData != nil {
		telemetryEndpoints = append(telemetryEndpoints, genesisData.TelemetryEndpoints...)
	} else {
		telemetryURLs := cfg.Global.TelemetryURLs
		for i := range telemetryURLs {
			telemetryEndpoints = append(telemetryEndpoints, &telemetryURLs[i])
		}
	}

	telemetryLogger := log.NewFromGlobal(log.AddContext("pkg", "telemetry"))
	return telemetry.BootstrapMailer(context.TODO(),
		telemetryEndpoints, !cfg.Global.NoTelemetry, telemetryLogger)
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
			logger.Errorf("failed to close database: %s", err)
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

func (n *nodeBuilder) loadRuntime(cfg *Config, ns *runtime.NodeStorage,
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

		rt, err := createRuntime(cfg, *ns, stateSrvc, ks, net, code)
		if err != nil {
			return err
		}

		runtimeCode[codeHash.String()] = rt
	}

	return nil
}
