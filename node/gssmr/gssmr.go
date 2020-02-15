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

package gssmr

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/config/genesis"
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/network"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/state"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

// Node is a container for all the components of a node.
type Node struct {
	Name      string
	Services  *services.ServiceRegistry // Registry of all core services
	RPC       *rpc.HTTPServer           // HTTP instance for RPC server
	IsStarted chan struct{}             // Signals node startup complete
	stop      chan struct{}             // Used to signal node shutdown
}

// NewNode initializes a Node with provided components.
func NewNode(name string, srvcs []services.Service, rpc *rpc.HTTPServer) *Node {
	d := &Node{
		Name:      name,
		Services:  services.NewServiceRegistry(),
		RPC:       rpc,
		IsStarted: make(chan struct{}),
		stop:      nil,
	}

	for _, srvc := range srvcs {
		d.Services.RegisterService(srvc)
	}

	return d
}

// Start starts all services. API service is started last.
func (d *Node) Start() {
	d.Services.StartAll()
	if d.RPC != nil {
		d.RPC.Start()
	}

	d.stop = make(chan struct{})
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		d.Stop()
		os.Exit(130)
	}()

	//Move on when routine catches SIGINT or SIGTERM calls
	close(d.IsStarted)
	d.Wait()
}

// Wait is used to force the node to stay alive until a signal is passed into `Node.stop`
func (d *Node) Wait() {
	<-d.stop
}

//Stop all services first, then send stop signal for test
func (d *Node) Stop() {
	d.Services.StopAll()
	if d.stop != nil {
		close(d.stop)
	}
}

// MakeNode sets up node; opening badgerDB instance and returning the Node container
func MakeNode(ctx *cli.Context, currentConfig *config.Config, ks *keystore.Keystore) (*Node, error) {
	var srvcs []services.Service

	dataDir := currentConfig.Global.DataDir

	// Create service, initialize stateDB and blockDB
	stateSrv := state.NewService(dataDir)
	srvcs = append(srvcs, stateSrv)

	err := stateSrv.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start db service: %s", err)
	}

	// Trie, runtime: load most recent state from DB, load runtime code from trie and create runtime executor
	rt, err := loadStateAndRuntime(stateSrv.Storage, ks)
	if err != nil {
		return nil, fmt.Errorf("error loading state and runtime: %s", err)
	}

	// load genesis from JSON file
	gendata, err := stateSrv.Storage.LoadGenesisData()
	if err != nil {
		return nil, err
	}

	// TODO: Configure node based on Roles #601

	// Network
	networkSrvc, networkMsgSend, networkMsgRec := createNetworkService(currentConfig, gendata, stateSrv)
	srvcs = append(srvcs, networkSrvc)

	// Core
	coreConfig := &core.Config{
		BlockState:    stateSrv.Block,
		StorageState:  stateSrv.Storage,
		Keystore:      ks,
		Runtime:       rt,
		MsgRec:        networkMsgSend, // message channel from network service to core service
		MsgSend:       networkMsgRec,  // message channel from core service to network service
		BabeAuthority: currentConfig.Global.Authority,
	}
	coreSrvc := createCoreService(coreConfig)
	srvcs = append(srvcs, coreSrvc)

	// API
	apiSrvc := api.NewAPIService(networkSrvc, nil)
	srvcs = append(srvcs, apiSrvc)

	return NewNode(gendata.Name, srvcs, nil), nil
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

// createNetworkService creates a network service from the command configuration and genesis data
func createNetworkService(fig *config.Config, gendata *genesis.GenesisData, stateService *state.Service) (*network.Service, chan network.Message, chan network.Message) {
	// Default bootnodes and protocol from genesis file
	bootnodes := common.BytesToStringArray(gendata.Bootnodes)
	protocolID := gendata.ProtocolID

	// If bootnodes flag has one or more bootnodes, overwrite genesis bootnodes
	if len(fig.Network.Bootnodes) > 0 {
		bootnodes = fig.Network.Bootnodes
	}

	// If protocol id flag is not an empty string, overwrite
	if fig.Network.ProtocolID != "" {
		protocolID = fig.Network.ProtocolID
	}

	// network service configuation
	networkConfig := network.Config{
		BlockState:   stateService.Block,
		StorageState: stateService.Storage,
		NetworkState: stateService.Network,
		DataDir:      fig.Global.DataDir,
		Roles:        fig.Global.Roles,
		Port:         fig.Network.Port,
		Bootnodes:    bootnodes,
		ProtocolID:   protocolID,
		NoBootstrap:  fig.Network.NoBootstrap,
		NoMdns:       fig.Network.NoMdns,
	}

	networkMsgRec := make(chan network.Message)
	networkMsgSend := make(chan network.Message)

	networkService, err := network.NewService(&networkConfig, networkMsgSend, networkMsgRec)
	if err != nil {
		log.Error("Failed to create new network service", "err", err)
	}

	return networkService, networkMsgSend, networkMsgRec
}

// createCoreService creates the core service from the provided core configuration
func createCoreService(coreConfig *core.Config) *core.Service {
	coreService, err := core.NewService(coreConfig)
	if err != nil {
		log.Error("Failed to create new core service", "err", err)
	}

	return coreService
}
