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

package cfg

import (
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
)

var (
	// P2P
	defaultP2PPort = 7001
	defaultP2PBoostrap = []string{"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", "/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",}
	defaultP2PRandSeed = int64(33)
	DefaultP2PConfig = &p2p.Config{
		Port: defaultP2PPort,
		RandSeed: defaultP2PRandSeed,
	}

	// DB
	DefaultDBConfig = &polkadb.Config{
		Datadir: DefaultDataDir(),
	}

	// RPC
	defaultRPCPort = uint32(8545)
	defaultRPCModules = []string{"core"}
	DefaultRPCConfig = &rpc.Config{
		Port: defaultRPCPort,
		// TODO: Need to add modules here or for API
	}
)


// Config is a collection of configurations throughout the system
type Config struct {
	P2PConfig 	*p2p.Config
	DbConfig    *polkadb.Config
	RPCConfig	*rpc.Config
}

// DefaultConfig is the default settings used when a config.toml file is not passed in during instantiation
var DefaultConfig = &Config{
	P2PConfig: DefaultP2PConfig,
	DbConfig: DefaultDBConfig,
	RPCConfig: DefaultRPCConfig,
}



//[Config]
//BootstrapNodes=["/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", "/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",]
//Port= 7001
//RandSeed= 33
//
//[DbConfig]
//Datadir="chaindata"




