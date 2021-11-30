// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChainSafe/chaindb"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/digest"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/pprof"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/life"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/utils"
)

func newInMemoryDB(path string) (chaindb.Database, error) {
	return utils.SetupDatabase(filepath.Join(path, "local_storage"), true)
}

// State Service

// createStateService creates the state service and initialise state database
func createStateService(cfg *Config) (*state.Service, error) {
	logger.Debug("creating state service...")

	config := state.Config{
		Path:     cfg.Global.BasePath,
		LogLevel: cfg.Log.StateLvl,
	}

	stateSrvc := state.NewService(config)

	// start state service (initialise state database)
	err := stateSrvc.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start state service: %s", err)
	}

	if cfg.State.Rewind != 0 {
		err = stateSrvc.Rewind(int64(cfg.State.Rewind))
		if err != nil {
			return nil, fmt.Errorf("failed to rewind state: %w", err)
		}
	}

	return stateSrvc, nil
}

func createRuntimeStorage(st *state.Service) (*runtime.NodeStorage, error) {
	localStorage, err := newInMemoryDB(st.DB().Path())
	if err != nil {
		return nil, err
	}

	return &runtime.NodeStorage{
		LocalStorage:      localStorage,
		PersistentStorage: chaindb.NewTable(st.DB(), "offlinestorage"),
		BaseDB:            st.Base,
	}, nil
}

func createRuntime(cfg *Config, ns runtime.NodeStorage, st *state.Service,
	ks *keystore.GlobalKeystore, net *network.Service, code []byte) (
	runtime.Instance, error) {
	logger.Info("creating runtime with interpreter " + cfg.Core.WasmInterpreter + "...")

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

	var rt runtime.Instance
	switch cfg.Core.WasmInterpreter {
	case wasmer.Name:
		rtCfg := &wasmer.Config{
			Imports: wasmer.ImportsNodeRuntime,
		}
		rtCfg.Storage = ts
		rtCfg.Keystore = ks
		rtCfg.LogLvl = cfg.Log.RuntimeLvl
		rtCfg.NodeStorage = ns
		rtCfg.Network = net
		rtCfg.Role = cfg.Core.Roles
		rtCfg.CodeHash = codeHash

		// create runtime executor
		rt, err = wasmer.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
	case life.Name:
		rtCfg := &life.Config{
			Resolver: new(life.Resolver),
		}
		rtCfg.Storage = ts
		rtCfg.Keystore = ks
		rtCfg.LogLvl = cfg.Log.RuntimeLvl
		rtCfg.NodeStorage = ns
		rtCfg.Network = net
		rtCfg.Role = cfg.Core.Roles
		rtCfg.CodeHash = codeHash

		// create runtime executor
		rt, err = life.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
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

func createBABEService(cfg *Config, st *state.Service, ks keystore.Keystore, cs *core.Service) (*babe.Service, error) {
	logger.Info("creating BABE service" +
		asAuthority(cfg.Core.BabeAuthority) + "...")

	if ks.Name() != "babe" || ks.Type() != crypto.Sr25519Type {
		return nil, ErrInvalidKeystoreType
	}

	kps := ks.Keypairs()
	logger.Infof("keystore with keys %v", kps)
	if len(kps) == 0 && cfg.Core.BabeAuthority {
		return nil, ErrNoKeysProvided
	}

	bcfg := &babe.ServiceConfig{
		LogLvl:             cfg.Log.BlockProducerLvl,
		BlockState:         st.Block,
		StorageState:       st.Storage,
		TransactionState:   st.Transaction,
		EpochState:         st.Epoch,
		BlockImportHandler: cs,
		Authority:          cfg.Core.BabeAuthority,
		IsDev:              cfg.Global.ID == "dev",
		Lead:               cfg.Core.BABELead,
	}

	if cfg.Core.BabeAuthority {
		bcfg.Keypair = kps[0].(*sr25519.Keypair)
	}

	// create new BABE service
	bs, err := babe.NewService(bcfg)
	if err != nil {
		logger.Errorf("failed to initialise BABE service: %s", err)
		return nil, err
	}

	return bs, nil
}

// Core Service

// createCoreService creates the core service from the provided core configuration
func createCoreService(cfg *Config, ks *keystore.GlobalKeystore,
	st *state.Service, net *network.Service, dh *digest.Handler) (
	*core.Service, error) {
	logger.Debug("creating core service" +
		asAuthority(cfg.Core.Roles == types.AuthorityRole) +
		"...")

	genesisData, err := st.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	codeSubs := make(map[common.Hash]string)
	for k, v := range genesisData.CodeSubstitutes {
		codeSubs[common.MustHexToHash(k)] = v
	}

	// set core configuration
	coreConfig := &core.Config{
		LogLvl:               cfg.Log.CoreLvl,
		BlockState:           st.Block,
		EpochState:           st.Epoch,
		StorageState:         st.Storage,
		TransactionState:     st.Transaction,
		Keystore:             ks,
		Network:              net,
		DigestHandler:        dh,
		CodeSubstitutes:      codeSubs,
		CodeSubstitutedState: st.Base,
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
func createNetworkService(cfg *Config, stateSrvc *state.Service) (*network.Service, error) {
	logger.Debugf(
		"creating network service with roles %d, port %d, bootnodes %s, protocol ID %s, nobootstrap=%t and noMDNS=%t...",
		cfg.Core.Roles, cfg.Network.Port, strings.Join(cfg.Network.Bootnodes, ","), cfg.Network.ProtocolID,
		cfg.Network.NoBootstrap, cfg.Network.NoMDNS)

	slotDuration, err := stateSrvc.Epoch.GetSlotDuration()
	if err != nil {
		return nil, fmt.Errorf("cannot get slot duration: %w", err)
	}

	// network service configuation
	networkConfig := network.Config{
		LogLvl:            cfg.Log.NetworkLvl,
		BlockState:        stateSrvc.Block,
		BasePath:          cfg.Global.BasePath,
		Roles:             cfg.Core.Roles,
		Port:              cfg.Network.Port,
		Bootnodes:         cfg.Network.Bootnodes,
		ProtocolID:        cfg.Network.ProtocolID,
		NoBootstrap:       cfg.Network.NoBootstrap,
		NoMDNS:            cfg.Network.NoMDNS,
		MinPeers:          cfg.Network.MinPeers,
		MaxPeers:          cfg.Network.MaxPeers,
		PublishMetrics:    cfg.Global.PublishMetrics,
		PersistentPeers:   cfg.Network.PersistentPeers,
		DiscoveryInterval: cfg.Network.DiscoveryInterval,
		SlotDuration:      slotDuration,
		PublicIP:          cfg.Network.PublicIP,
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
func createRPCService(cfg *Config, ns *runtime.NodeStorage, stateSrvc *state.Service,
	coreSrvc *core.Service, networkSrvc *network.Service, bp modules.BlockProducerAPI,
	sysSrvc *system.Service, finSrvc *grandpa.Service) (*rpc.HTTPServer, error) {
	logger.Infof(
		"creating rpc service with host %s, external=%t, port %d, modules %s, ws=%t, ws port %d and ws external=%t",
		cfg.RPC.Host, cfg.RPC.External, cfg.RPC.Port, strings.Join(cfg.RPC.Modules, ","), cfg.RPC.WS,
		cfg.RPC.WSPort, cfg.RPC.WSExternal,
	)
	rpcService := rpc.NewService()

	genesisData, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, fmt.Errorf("failed to load genesis data: %s", err)
	}

	syncStateSrvc, err := modules.NewStateSync(genesisData, stateSrvc.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync state service: %s", err)
	}

	rpcConfig := &rpc.HTTPServerConfig{
		LogLvl:              cfg.Log.RPCLvl,
		BlockAPI:            stateSrvc.Block,
		StorageAPI:          stateSrvc.Storage,
		NetworkAPI:          networkSrvc,
		CoreAPI:             coreSrvc,
		NodeStorage:         ns,
		BlockProducerAPI:    bp,
		BlockFinalityAPI:    finSrvc,
		TransactionQueueAPI: stateSrvc.Transaction,
		RPCAPI:              rpcService,
		SyncStateAPI:        syncStateSrvc,
		SystemAPI:           sysSrvc,
		RPC:                 cfg.RPC.Enabled,
		RPCExternal:         cfg.RPC.External,
		RPCUnsafe:           cfg.RPC.Unsafe,
		RPCUnsafeExternal:   cfg.RPC.UnsafeExternal,
		Host:                cfg.RPC.Host,
		RPCPort:             cfg.RPC.Port,
		WS:                  cfg.RPC.WS,
		WSExternal:          cfg.RPC.WSExternal,
		WSUnsafe:            cfg.RPC.WSUnsafe,
		WSUnsafeExternal:    cfg.RPC.WSUnsafeExternal,
		WSPort:              cfg.RPC.WSPort,
		Modules:             cfg.RPC.Modules,
	}

	return rpc.NewHTTPServer(rpcConfig), nil
}

// createSystemService creates a systemService for providing system related information
func createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
	genesisData, err := stateSrvc.Base.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	return system.NewService(cfg, genesisData), nil
}

// createGRANDPAService creates a new GRANDPA service
func createGRANDPAService(cfg *Config, st *state.Service, dh *digest.Handler,
	ks keystore.Keystore, net *network.Service) (*grandpa.Service, error) {
	rt, err := st.Block.GetRuntime(nil)
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
	if len(keys) == 0 && cfg.Core.GrandpaAuthority {
		return nil, errors.New("no ed25519 keys provided for GRANDPA")
	}

	gsCfg := &grandpa.Config{
		LogLvl:        cfg.Log.FinalityGadgetLvl,
		BlockState:    st.Block,
		GrandpaState:  st.Grandpa,
		DigestHandler: dh,
		Voters:        voters,
		Authority:     cfg.Core.GrandpaAuthority,
		Network:       net,
		Interval:      cfg.Core.GrandpaInterval,
	}

	if cfg.Core.GrandpaAuthority {
		gsCfg.Keypair = keys[0].(*ed25519.Keypair)
	}

	return grandpa.NewService(gsCfg)
}

func createBlockVerifier(st *state.Service) (*babe.VerificationManager, error) {
	ver, err := babe.NewVerificationManager(st.Block, st.Epoch)
	if err != nil {
		return nil, err
	}

	return ver, nil
}

func newSyncService(cfg *Config, st *state.Service, fg sync.FinalityGadget,
	verifier *babe.VerificationManager, cs *core.Service, net *network.Service) (
	*sync.Service, error) {
	slotDuration, err := st.Epoch.GetSlotDuration()
	if err != nil {
		return nil, err
	}

	syncCfg := &sync.Config{
		LogLvl:             cfg.Log.SyncLvl,
		Network:            net,
		BlockState:         st.Block,
		StorageState:       st.Storage,
		TransactionState:   st.Transaction,
		FinalityGadget:     fg,
		BabeVerifier:       verifier,
		BlockImportHandler: cs,
		MinPeers:           cfg.Network.MinPeers,
		MaxPeers:           cfg.Network.MaxPeers,
		SlotDuration:       slotDuration,
	}

	return sync.NewService(syncCfg)
}

func createDigestHandler(st *state.Service) (*digest.Handler, error) {
	return digest.NewHandler(st.Block, st.Epoch, st.Grandpa)
}

func createPprofService(settings pprof.Settings) (service *pprof.Service) {
	pprofLogger := log.NewFromGlobal(log.AddContext("pkg", "pprof"))
	return pprof.NewService(settings, pprofLogger)
}
