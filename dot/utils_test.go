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
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
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

func TestNewTestGenesisHRFile(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genHRFile := NewTestGenesisHRFile(t, cfg)
	require.NotNil(t, genHRFile)
	defer os.Remove(genHRFile.Name())

	genRawFile := NewTestGenesisFile(t, cfg)
	require.NotNil(t, genRawFile)
	defer os.Remove(genRawFile.Name())

	genHR, err := genesis.NewGenesisFromJSONHR(genHRFile.Name())
	require.NoError(t, err)
	genRaw, err := genesis.NewGenesisFromJSON(genRawFile.Name())
	require.NoError(t, err)

	// values from raw genesis file should equal values generated from human readable genesis file
	//  Note, we don't just compare Genesis.Raw because there are entries (balances) in raw genesis that are not yet handled by
	//  by human readable genesis yet.
	require.Equal(t, genRaw.Genesis.Raw[0]["0x3a636f6465"], genHR.Genesis.Raw[0]["0x3a636f6465"])                                                             // check system code entry
	require.Equal(t, genRaw.Genesis.Raw[0]["0x3a6772616e6470615f617574686f726974696573"], genHR.Genesis.Raw[0]["0x3a6772616e6470615f617574686f726974696573"]) // grandpa authority entry
	require.Equal(t, genRaw.Genesis.Raw[0]["0x886726f904d8372fdabb7707870c2fad"], genHR.Genesis.Raw[0]["0x886726f904d8372fdabb7707870c2fad"])                 // check babe authorities entry

}
