// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package rpc

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestNewService(t *testing.T) {
	ctx := &cli.Context{
		App: &cli.App{
			Name:    "gossamer",
			Version: "0.0.1",
		},
	}
	NewService(ctx, "gssmr")
}

func TestService_Methods(t *testing.T) {
	qtySystemMethods := 7
	qtyRPCMethods := 1
	qtyAuthorMethods := 6

	ctx := &cli.Context{
		App: &cli.App{
			Name:    "gossamer",
			Version: "0.0.1",
		},
	}
	rpcService := NewService(ctx, "gssmr")
	sysMod := modules.NewSystemModule(nil, nil)
	rpcService.BuildMethodNames(sysMod, "system")
	m := rpcService.Methods()
	require.Equal(t, qtySystemMethods, len(m)) // check to confirm quantity for methods is correct

	rpcMod := modules.NewRPCModule(nil)
	rpcService.BuildMethodNames(rpcMod, "rpc")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods, len(m))

	authMod := modules.NewAuthorModule(nil, nil, nil)
	rpcService.BuildMethodNames(authMod, "author")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods+qtyAuthorMethods, len(m))
}

func TestService_NodeName(t *testing.T) {
	ctx := &cli.Context{
		App: &cli.App{
			Name:    "gossamer",
			Version: "0.0.1",
		},
	}
	rpcService := NewService(ctx, "gssmr")

	name := rpcService.NodeName()
	require.Equal(t, "gssmr", name)
}

func TestService_SystemName(t *testing.T) {
	ctx := &cli.Context{
		App: &cli.App{
			Name:    "gossamer",
			Version: "0.0.1",
		},
	}
	rpcService := NewService(ctx, "gssmr")

	name := rpcService.SystemName()
	require.Equal(t, "gossamer", name)
}

func TestService_SystemVersion(t *testing.T) {
	ctx := &cli.Context{
		App: &cli.App{
			Name:    "gossamer",
			Version: "0.0.1",
		},
	}
	rpcService := NewService(ctx, "gssmr")

	ver := rpcService.SystemVersion()
	require.Equal(t, "0.0.1", ver)
}
