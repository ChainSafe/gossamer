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
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ChainSafe/chaindb"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/rpc"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/life"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmtime"
)

func newInMemoryDB(path string) (chaindb.Database, error) {
	return chaindb.NewBadgerDB(&chaindb.Config{
		DataDir:  filepath.Join(path, "local_storage"),
		InMemory: true,
	})
}

// State Service

// createStateService creates the state service and initialize state database
func createStateService(cfg *Config) (*state.Service, error) {
	logger.Debug("creating state service...")
	stateSrvc := state.NewService(cfg.Global.BasePath, cfg.Log.StateLvl)

	// start state service (initialize state database)
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

	// load most recent state from database
	latestState, err := state.LoadLatestStorageHash(stateSrvc.DB())
	if err != nil {
		return nil, fmt.Errorf("failed to load latest state root hash: %s", err)
	}

	// load most recent state from database
	_, err = stateSrvc.Storage.LoadFromDB(latestState)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest state from database: %s", err)
	}

	return stateSrvc, nil
}

func createRuntime(cfg *Config, st *state.Service, ks *keystore.GlobalKeystore, net *network.Service) (runtime.Instance, error) {
	logger.Info(
		"creating runtime...",
		"interpreter", cfg.Core.WasmInterpreter,
	)

	// load runtime code from trie
	code, err := st.Storage.GetStorage(nil, []byte(":code"))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve :code from trie: %s", err)
	}

	ts, err := st.Storage.TrieState(nil)
	if err != nil {
		return nil, err
	}

	localStorage, err := newInMemoryDB(st.DB().Path())
	if err != nil {
		return nil, err
	}

	ns := runtime.NodeStorage{
		LocalStorage:      localStorage,
		PersistentStorage: chaindb.NewTable(st.DB(), "offlinestorage"),
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

		// create runtime executor
		rt, err = wasmer.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
	case wasmtime.Name:
		rtCfg := &wasmtime.Config{
			Imports: wasmtime.ImportNodeRuntime,
		}
		rtCfg.Storage = ts
		rtCfg.Keystore = ks
		rtCfg.LogLvl = cfg.Log.RuntimeLvl
		rtCfg.NodeStorage = ns
		rtCfg.Network = net
		rtCfg.Role = cfg.Core.Roles

		// create runtime executor
		rt, err = wasmtime.NewInstance(code, rtCfg)
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

		// create runtime executor
		rt, err = life.NewInstance(code, rtCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime executor: %s", err)
		}
	}

	return rt, nil
}

func createBABEService(cfg *Config, rt runtime.Instance, st *state.Service, ks keystore.Keystore) (*babe.Service, error) {
	logger.Info(
		"creating BABE service...",
		"authority", cfg.Core.BabeAuthority,
	)

	if ks.Name() != "babe" || ks.Type() != crypto.Sr25519Type {
		return nil, ErrInvalidKeystoreType
	}

	kps := ks.Keypairs()
	logger.Info("keystore", "keys", kps)
	if len(kps) == 0 && cfg.Core.BabeAuthority {
		return nil, ErrNoKeysProvided
	}

	bcfg := &babe.ServiceConfig{
		LogLvl:               cfg.Log.BlockProducerLvl,
		Runtime:              rt,
		BlockState:           st.Block,
		StorageState:         st.Storage,
		TransactionState:     st.Transaction,
		EpochState:           st.Epoch,
		EpochLength:          cfg.Core.EpochLength,
		ThresholdNumerator:   cfg.Core.BabeThresholdNumerator,
		ThresholdDenominator: cfg.Core.BabeThresholdDenominator,
		SlotDuration:         cfg.Core.SlotDuration,
		Authority:            cfg.Core.BabeAuthority,
	}

	if cfg.Core.BabeAuthority {
		bcfg.Keypair = kps[0].(*sr25519.Keypair)
	}

	// create new BABE service
	bs, err := babe.NewService(bcfg)
	if err != nil {
		logger.Error("failed to initialize BABE service", "error", err)
		return nil, err
	}

	return bs, nil
}

// Core Service

// createCoreService creates the core service from the provided core configuration
func createCoreService(cfg *Config, bp core.BlockProducer, fg core.FinalityGadget, verifier *babe.VerificationManager, rt runtime.Instance, ks *keystore.GlobalKeystore, stateSrvc *state.Service, net *network.Service) (*core.Service, error) {
	logger.Debug(
		"creating core service...",
		"authority", cfg.Core.Roles == types.AuthorityRole,
	)

	// set core configuration
	coreConfig := &core.Config{
		LogLvl:              cfg.Log.CoreLvl,
		BlockState:          stateSrvc.Block,
		EpochState:          stateSrvc.Epoch,
		StorageState:        stateSrvc.Storage,
		TransactionState:    stateSrvc.Transaction,
		BlockProducer:       bp,
		FinalityGadget:      fg,
		Keystore:            ks,
		Runtime:             rt,
		IsBlockProducer:     cfg.Core.BabeAuthority,
		IsFinalityAuthority: cfg.Core.GrandpaAuthority,
		Verifier:            verifier,
		Network:             net,
	}

	// create new core service
	coreSrvc, err := core.NewService(coreConfig)
	if err != nil {
		logger.Error("failed to create core service", "error", err)
		return nil, err
	}

	return coreSrvc, nil
}

// Network Service

// createNetworkService creates a network service from the command configuration and genesis data
func createNetworkService(cfg *Config, stateSrvc *state.Service) (*network.Service, error) {
	logger.Debug(
		"creating network service...",
		"roles", cfg.Core.Roles,
		"port", cfg.Network.Port,
		"bootnodes", cfg.Network.Bootnodes,
		"protocol", cfg.Network.ProtocolID,
		"nobootstrap", cfg.Network.NoBootstrap,
		"nomdns", cfg.Network.NoMDNS,
	)

	// network service configuation
	networkConfig := network.Config{
		LogLvl:          cfg.Log.NetworkLvl,
		BlockState:      stateSrvc.Block,
		BasePath:        cfg.Global.BasePath,
		Roles:           cfg.Core.Roles,
		Port:            cfg.Network.Port,
		Bootnodes:       cfg.Network.Bootnodes,
		ProtocolID:      cfg.Network.ProtocolID,
		NoBootstrap:     cfg.Network.NoBootstrap,
		NoMDNS:          cfg.Network.NoMDNS,
		MinPeers:        cfg.Network.MinPeers,
		MaxPeers:        cfg.Network.MaxPeers,
		PublishMetrics:  cfg.Global.PublishMetrics,
		PersistentPeers: cfg.Network.PersistentPeers,
	}

	networkSrvc, err := network.NewService(&networkConfig)
	if err != nil {
		logger.Error("failed to create network service", "error", err)
		return nil, err
	}

	return networkSrvc, nil
}

// RPC Service

// createRPCService creates the RPC service from the provided core configuration
func createRPCService(cfg *Config, stateSrvc *state.Service, coreSrvc *core.Service, networkSrvc *network.Service, bp modules.BlockProducerAPI, rt runtime.Instance, sysSrvc *system.Service) *rpc.HTTPServer {
	logger.Info(
		"creating rpc service...",
		"host", cfg.RPC.Host,
		"external", cfg.RPC.External,
		"rpc port", cfg.RPC.Port,
		"mods", cfg.RPC.Modules,
		"ws", cfg.RPC.WS,
		"ws port", cfg.RPC.WSPort,
		"ws external", cfg.RPC.WSExternal,
	)
	rpcService := rpc.NewService()

	rpcConfig := &rpc.HTTPServerConfig{
		LogLvl:              cfg.Log.RPCLvl,
		BlockAPI:            stateSrvc.Block,
		StorageAPI:          stateSrvc.Storage,
		NetworkAPI:          networkSrvc,
		CoreAPI:             coreSrvc,
		BlockProducerAPI:    bp,
		RuntimeAPI:          rt,
		TransactionQueueAPI: stateSrvc.Transaction,
		RPCAPI:              rpcService,
		SystemAPI:           sysSrvc,
		External:            cfg.RPC.External,
		Host:                cfg.RPC.Host,
		RPCPort:             cfg.RPC.Port,
		WS:                  cfg.RPC.WS,
		WSExternal:          cfg.RPC.WSExternal,
		WSPort:              cfg.RPC.WSPort,
		Modules:             cfg.RPC.Modules,
	}

	return rpc.NewHTTPServer(rpcConfig)
}

// System service
// creates a service for providing system related information
func createSystemService(cfg *types.SystemInfo, stateSrvc *state.Service) (*system.Service, error) {
	genesisData, err := stateSrvc.Storage.GetGenesisData()
	if err != nil {
		return nil, err
	}
	// TODO: use data from genesisData for SystemInfo once they are in database (See issue #1248)
	return system.NewService(cfg, genesisData), nil
}

// createGRANDPAService creates a new GRANDPA service
func createGRANDPAService(cfg *Config, rt runtime.Instance, st *state.Service, dh *core.DigestHandler, ks keystore.Keystore, net *network.Service) (*grandpa.Service, error) {
	ad, err := rt.GrandpaAuthorities()
	if err != nil {
		return nil, err
	}

	if ks.Name() != "gran" || ks.Type() != crypto.Ed25519Type {
		return nil, ErrInvalidKeystoreType
	}

	voters := grandpa.NewVotersFromAuthorities(ad)

	keys := ks.Keypairs()
	if len(keys) == 0 && cfg.Core.GrandpaAuthority {
		return nil, errors.New("no ed25519 keys provided for GRANDPA")
	}

	gsCfg := &grandpa.Config{
		LogLvl:        cfg.Log.FinalityGadgetLvl,
		BlockState:    st.Block,
		DigestHandler: dh,
		SetID:         1,
		Voters:        voters,
		Authority:     cfg.Core.GrandpaAuthority,
		Network:       net,
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

func createSyncService(cfg *Config, st *state.Service, bp sync.BlockProducer, dh *core.DigestHandler, verifier *babe.VerificationManager, rt runtime.Instance) (*sync.Service, error) {
	syncCfg := &sync.Config{
		LogLvl:           cfg.Log.SyncLvl,
		BlockState:       st.Block,
		StorageState:     st.Storage,
		TransactionState: st.Transaction,
		BlockProducer:    bp,
		Verifier:         verifier,
		Runtime:          rt,
		DigestHandler:    dh,
	}

	return sync.NewService(syncCfg)
}

func createDigestHandler(st *state.Service, bp core.BlockProducer, verifier *babe.VerificationManager) (*core.DigestHandler, error) {
	return core.NewDigestHandler(st.Block, st.Epoch, bp, nil, verifier)
}
