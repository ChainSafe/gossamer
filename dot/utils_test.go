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
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestNewConfig tests the NewTestConfig method
func TestNewConfig(t *testing.T) {
	cfg := NewTestConfig(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
	require.NotNil(t, cfg)
}

// TestNewConfigAndFile tests the NewTestConfigWithFile method
func TestNewConfigAndFile(t *testing.T) {
	testCfg, testCfgFile := NewTestConfigWithFile(t)

	defer utils.RemoveTestDir(t)

	// TODO: improve dot tests #687
	require.NotNil(t, testCfg)
	require.NotNil(t, testCfgFile)
}

// TestInitNode
func TestNewTestGenesis(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genFile)

	defer utils.RemoveTestDir(t)

	cfg.Init.Genesis = genFile.Name()
	fmt.Printf("FileName %v\n", genFile.Name())
}

func TestNewTestGenesisFromJSONHR(t *testing.T) {
	//gen, err := genesis.NewGenesisFromJSONHR("../gossamer_genesis.json")
	gen, err := genesis.NewGenesisFromJSONHR("../myCustomSpec.json")
	require.NoError(t, err)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	fmt.Printf("alice address %v\n", kr.Alice.Public().Address())
	fmt.Printf("alice hex %x\n", kr.Alice.Public().Encode())
	npk, err := ed25519.NewPublicKey(kr.Alice.Public().Encode())
	require.NoError(t, err)
	fmt.Printf("npk %v\n", npk.Address())
	fmt.Printf("gen %v\n", gen.Name)
}