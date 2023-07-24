// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"fmt"
	"strings"

	cfg "github.com/ChainSafe/gossamer/config"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/metrics"
	"github.com/ChainSafe/gossamer/internal/pprof"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/utils"
)

// BlockProducer to produce blocks
type BlockProducer interface {
	Pause() error
	Resume() error
	EpochLength() uint64
	SlotDuration() uint64
}

type rpcServiceSettings struct {
	config        *cfg.Config
	nodeStorage   *runtime.NodeStorage
	state         *state.Service
	core          *core.Service
	network       *network.Service
	blockProducer BlockProducer
	system        *system.Service
	blockFinality *grandpa.Service
	syncer        *sync.Service
}

func newInMemoryDB() (database.Database, error) {
	return utils.SetupDatabase("", true)
}

// createStateService creates the state service and initialise state database
func (nodeBuilder) createStateService(config *cfg.Config) (*state.Service, error) {
	logger.Debug("creating state service...")

	stateLogLevel, err := log.ParseLevel(config.Log.State)
	if err != nil {
		return nil, err
	}
	stateConfig := state.Config{
		Path:     config.BasePath,
		LogLevel: stateLogLevel,
		Metrics:  metrics.NewIntervalConfig(config.PrometheusExternal),
	}

	stateSrvc := state.NewService(stateConfig)

	if err := stateSrvc.SetupBase(); err != nil {
		return nil, fmt.Errorf("cannot setup base: %w", err)
	}

	return stateSrvc, nil
}

func startStateService(config cfg.StateConfig, stateSrvc *state.Service) error {
	logger.Debug("starting state service...")

	// start state service (initialise state database)
	err := stateSrvc.Start()
	if err != nil {
		return fmt.Errorf("failed to start state service: %w", err)
	}

	if config.Rewind != 0 {
		err = stateSrvc.Rewind(config.Rewind)
		if err != nil {
			return fmt.Errorf("failed to rewind state: %w", err)
		}
	}

	return nil
}

func (nodeBuilder) createRuntimeStorage(st *state.Service) (*runtime.NodeStorage, error) {
	localStorage, err := newInMemoryDB()
	if err != nil {
		return nil, err
	}

	return &runtime.NodeStorage{
		LocalStorage:      localStorage,
		PersistentStorage: database.NewTable(st.DB(), "offlinestorage"),
		BaseDB:            st.Base,
	}, nil
}

func createRuntime(config *cfg.Config, ns runtime.NodeStorage, st *state.Service,
	ks *keystore.GlobalKeystore, net *network.Service, code []byte) (
	rt runtime.Instance, err error) {
	logger.Info("creating runtime with interpreter " + config.Core.WasmInterpreter + "...")

	// check if code substitute is in use, if so replace code
	codeSubHash := st.Base.LoadCodeSubstitutedBlockHash()

	if !codeSubHash.IsEmpty() {
		logger.Infof("🔄 detected runtime code substitution, upgrading to block hash %s...", codeSubHash)
		genData, err := st.Base.LoadGenesisData()
		if err != nil {
			return nil, err
		}
		codeString := genData.CodeSubstitutes[codeSubHash.String()]

		code = common.MustHexToBytes(codeString)
	}

	ts, err := st.Storage.TrieState(nil)
	if err != nil {
		return nil, err
	}

	codeHash, err := st.Storage.LoadCodeHash(nil)
	if err != nil {
		return nil, err
	}

	wasmerLogLevel, err := log.ParseLevel(config.Log.Wasmer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wasmer log level: %w", err)
	}
	switch config.Core.WasmInterpreter {
	case wasmer.Name:
		rtCfg := wasmer.Config{
			Storage:     ts,
			Keystore:    ks,
			LogLvl:      wasmerLogLevel,
			NodeStorage: ns,
			Network:     net,
			Role:        config.Core.Role,
			CodeHash:    codeHash,
		}

		// create runtime executor
		rt, err = wasmer.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
	case wazero_runtime.Name:
		rtCfg := wazero_runtime.Config{
			Storage:     ts,
			Keystore:    ks,
			LogLvl:      wasmerLogLevel,
			NodeStorage: ns,
			Network:     net,
			Role:        config.Core.Role,
			CodeHash:    codeHash,
		}

		// create runtime executor
		rt, err = wazero_runtime.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrWasmInterpreterName, config.Core.WasmInterpreter)
	}

	st.Block.StoreRuntime(st.Block.BestBlockHash(), rt)
	return rt, nil
}

func asAuthority(authority bool) string {
	if authority {
		return " as authority"
	}
	return ""
}

// ServiceBuilder interface to define the building of babe service
type ServiceBuilder interface {
	NewServiceIFace(cfg *babe.ServiceConfig) (service *babe.Service, err error)
}

var _ ServiceBuilder = (*babe.Builder)(nil)

func (nb nodeBuilder) createBABEService(config *cfg.Config, st *state.Service, ks KeyStore,
	cs *core.Service, telemetryMailer Telemetry) (service *babe.Service, err error) {
	return nb.createBABEServiceWithBuilder(config, st, ks, cs, telemetryMailer, babe.Builder{})
}

// KeyStore is the keystore interface for the BABE service.
type KeyStore interface {
	Name() keystore.Name
	Type() string
	Keypairs() []keystore.KeyPair
}

func (nodeBuilder) createBABEServiceWithBuilder(config *cfg.Config, st *state.Service, ks KeyStore,
	cs *core.Service, telemetryMailer Telemetry, newBabeService ServiceBuilder) (
	service *babe.Service, err error) {
	logger.Info("creating BABE service" +
		asAuthority(config.Core.BabeAuthority) + "...")

	if ks.Name() != "babe" || ks.Type() != crypto.Sr25519Type {
		return nil, ErrInvalidKeystoreType
	}

	kps := ks.Keypairs()
	logger.Infof("keystore with keys %v", kps)
	if len(kps) == 0 && config.Core.BabeAuthority {
		return nil, ErrNoKeysProvided
	}

	babeLogLevel, err := log.ParseLevel(config.Log.Babe)
	if err != nil {
		return nil, fmt.Errorf("failed to parse babe log level: %w", err)
	}
	bcfg := &babe.ServiceConfig{
		LogLvl:             babeLogLevel,
		BlockState:         st.Block,
		StorageState:       st.Storage,
		TransactionState:   st.Transaction,
		EpochState:         st.Epoch,
		BlockImportHandler: cs,
		Authority:          config.Core.BabeAuthority,
		IsDev:              config.ID == "dev",
		Telemetry:          telemetryMailer,
	}

	if config.Core.BabeAuthority {
		bcfg.Keypair = kps[0].(*sr25519.Keypair)
	}

	bs, err := newBabeService.NewServiceIFace(bcfg)
	if err != nil {
		logger.Errorf("failed to initialise BABE service: %s", err)
		return nil, err
	}
	return bs, nil
}

// Core Service

// createCoreService creates the core service from the provided core configuration
func (nodeBuilder) createCoreService(config *cfg.Config, ks *keystore.GlobalKeystore,
	st *state.Service, net *network.Service, dh *digest.Handler) (
	*core.Service, error) {
	logger.Debug("creating core service" +
		asAuthority(config.Core.Role == common.AuthorityRole) +
		"...")

	genesisData, err := st.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	codeSubs := make(map[common.Hash]string)
	for k, v := range genesisData.CodeSubstitutes {
		codeSubs[common.MustHexToHash(k)] = v
	}

	coreLogLevel, err := log.ParseLevel(config.Log.Core)
	if err != nil {
		return nil, fmt.Errorf("failed to parse core log level: %w", err)
	}
	// set core configuration
	coreConfig := &core.Config{
		LogLvl:               coreLogLevel,
		BlockState:           st.Block,
		StorageState:         st.Storage,
		TransactionState:     st.Transaction,
		Keystore:             ks,
		Network:              net,
		CodeSubstitutes:      codeSubs,
		CodeSubstitutedState: st.Base,
		OnBlockImport:        digest.NewBlockImportHandler(st.Epoch, st.Grandpa),
	}

	// create new core service
	coreSrvc, err := core.NewService(coreConfig)
	if err != nil {
		logger.Errorf("failed to create core service: %s", err)
		return nil, err
	}

	return coreSrvc, nil
}

// Network Service

// createNetworkService creates a network service from the command configuration and genesis data
func (nodeBuilder) createNetworkService(config *cfg.Config, stateSrvc *state.Service,
	telemetryMailer Telemetry) (*network.Service, error) {
	logger.Debugf(
		"creating network service with role %d, port %d, bootnodes %s, protocol ID %s, nobootstrap=%t and noMDNS=%t...",
		config.Core.Role, config.Network.Port, strings.Join(config.Network.Bootnodes, ","), config.Network.ProtocolID,
		config.Network.NoBootstrap, config.Network.NoMDNS)

	slotDuration, err := stateSrvc.Epoch.GetSlotDuration()
	if err != nil {
		return nil, fmt.Errorf("cannot get slot duration: %w", err)
	}

	networkLogLevel, err := log.ParseLevel(config.Log.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to parse network log level: %w", err)
	}
	// network service configuation
	networkConfig := network.Config{
		LogLvl:            networkLogLevel,
		BlockState:        stateSrvc.Block,
		BasePath:          config.BasePath,
		Roles:             config.Core.Role,
		Port:              config.Network.Port,
		Bootnodes:         config.Network.Bootnodes,
		ProtocolID:        config.Network.ProtocolID,
		NoBootstrap:       config.Network.NoBootstrap,
		NoMDNS:            config.Network.NoMDNS,
		MinPeers:          config.Network.MinPeers,
		MaxPeers:          config.Network.MaxPeers,
		PersistentPeers:   config.Network.PersistentPeers,
		DiscoveryInterval: config.Network.DiscoveryInterval,
		SlotDuration:      slotDuration,
		PublicIP:          config.Network.PublicIP,
		Telemetry:         telemetryMailer,
		PublicDNS:         config.Network.PublicDNS,
		Metrics:           metrics.NewIntervalConfig(config.PrometheusExternal),
		NodeKey:           config.Network.NodeKey,
		ListenAddress:     config.Network.ListenAddress,
	}

	networkSrvc, err := network.NewService(&networkConfig)
	if err != nil {
		logger.Errorf("failed to create network service: %s", err)
		return nil, err
	}

	return networkSrvc, nil
}

// RPC Service

// createRPCService creates the RPC service from the provided core configuration
func (nodeBuilder) createRPCService(params rpcServiceSettings) (*rpc.HTTPServer, error) {
	logger.Infof(
		"creating rpc service with host %s, external=%t, port %d, modules %s, ws port %d and ws external=%t",
		params.config.RPC.Host,
		params.config.RPC.RPCExternal,
		params.config.RPC.Port,
		strings.Join(params.config.RPC.Modules, ","),
		params.config.RPC.WSPort,
		params.config.RPC.WSExternal,
	)
	rpcService := rpc.NewService()

	genesisData, err := params.state.Base.LoadGenesisData()
	if err != nil {
		return nil, fmt.Errorf("failed to load genesis data: %s", err)
	}

	syncStateSrvc, err := modules.NewStateSync(genesisData, params.state.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync state service: %s", err)
	}

	rpcLogLevel, err := log.ParseLevel(params.config.Log.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rpc log level: %w", err)
	}
	rpcConfig := &rpc.HTTPServerConfig{
		LogLvl:              rpcLogLevel,
		BlockAPI:            params.state.Block,
		StorageAPI:          params.state.Storage,
		NetworkAPI:          params.network,
		CoreAPI:             params.core,
		NodeStorage:         params.nodeStorage,
		BlockProducerAPI:    params.blockProducer,
		BlockFinalityAPI:    params.blockFinality,
		TransactionQueueAPI: params.state.Transaction,
		RPCAPI:              rpcService,
		SyncStateAPI:        syncStateSrvc,
		SyncAPI:             params.syncer,
		SystemAPI:           params.system,
		RPCUnsafe:           params.config.RPC.UnsafeRPC,
		RPCExternal:         params.config.RPC.RPCExternal,
		RPCUnsafeExternal:   params.config.RPC.UnsafeRPCExternal,
		Host:                params.config.RPC.Host,
		RPCPort:             params.config.RPC.Port,
		WSExternal:          params.config.RPC.WSExternal,
		WSUnsafeExternal:    params.config.RPC.UnsafeWSExternal,
		WSPort:              params.config.RPC.WSPort,
		Modules:             params.config.RPC.Modules,
	}

	return rpc.NewHTTPServer(rpcConfig), nil
}

// createSystemService creates a systemService for providing system related information
func (nodeBuilder) createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
	genesisData, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	return system.NewService(cfg, genesisData), nil
}

// createGRANDPAService creates a new GRANDPA service
func (nodeBuilder) createGRANDPAService(config *cfg.Config, st *state.Service, ks KeyStore,
	net *network.Service, telemetryMailer Telemetry) (*grandpa.Service, error) {
	bestBlockHash := st.Block.BestBlockHash()
	rt, err := st.Block.GetRuntime(bestBlockHash)
	if err != nil {
		return nil, err
	}

	ad, err := rt.GrandpaAuthorities()
	if err != nil {
		return nil, err
	}

	if ks.Name() != "gran" || ks.Type() != crypto.Ed25519Type {
		return nil, ErrInvalidKeystoreType
	}

	voters := types.NewGrandpaVotersFromAuthorities(ad)

	keys := ks.Keypairs()
	if len(keys) == 0 && config.Core.GrandpaAuthority {
		return nil, errors.New("no ed25519 keys provided for GRANDPA")
	}

	grandpaLogLevel, err := log.ParseLevel(config.Log.Grandpa)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grandpa log level: %w", err)
	}
	gsCfg := &grandpa.Config{
		LogLvl:       grandpaLogLevel,
		BlockState:   st.Block,
		GrandpaState: st.Grandpa,
		Voters:       voters,
		Authority:    config.Core.GrandpaAuthority,
		Network:      net,
		Interval:     config.Core.GrandpaInterval,
		Telemetry:    telemetryMailer,
	}

	if config.Core.GrandpaAuthority {
		gsCfg.Keypair = keys[0].(*ed25519.Keypair)
	}

	return grandpa.NewService(gsCfg)
}

func (nodeBuilder) createBlockVerifier(st *state.Service) *babe.VerificationManager {
	return babe.NewVerificationManager(st.Block, st.Slot, st.Epoch)
}

func (nodeBuilder) newSyncService(config *cfg.Config, st *state.Service, fg BlockJustificationVerifier,
	verifier *babe.VerificationManager, cs *core.Service, net *network.Service, telemetryMailer Telemetry) (
	*sync.Service, error) {
	slotDuration, err := st.Epoch.GetSlotDuration()
	if err != nil {
		return nil, err
	}

	genesisData, err := st.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	syncLogLevel, err := log.ParseLevel(config.Log.Sync)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sync log level: %w", err)
	}
	syncCfg := &sync.Config{
		LogLvl:             syncLogLevel,
		Network:            net,
		BlockState:         st.Block,
		StorageState:       st.Storage,
		TransactionState:   st.Transaction,
		FinalityGadget:     fg,
		BabeVerifier:       verifier,
		BlockImportHandler: cs,
		MinPeers:           config.Network.MinPeers,
		MaxPeers:           config.Network.MaxPeers,
		SlotDuration:       slotDuration,
		Telemetry:          telemetryMailer,
		BadBlocks:          genesisData.BadBlocks,
	}

	blockReqRes := net.GetRequestResponseProtocol(network.SyncID, network.BlockRequestTimeout,
		network.MaxBlockResponseSize)

	return sync.NewService(syncCfg, blockReqRes)
}

func (nodeBuilder) createDigestHandler(st *state.Service) (*digest.Handler, error) {
	return digest.NewHandler(st.Block, st.Epoch, st.Grandpa)
}

func createPprofService(config cfg.PprofConfig) (service *pprof.Service) {
	pprofLogger := log.NewFromGlobal(log.AddContext("pkg", "pprof"))
	return pprof.NewService(config, pprofLogger)
}
