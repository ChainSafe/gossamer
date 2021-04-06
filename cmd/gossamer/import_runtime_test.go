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

package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"

	"github.com/stretchr/testify/require"
)

func TestCreateGenesisWithRuntime(t *testing.T) {
	defaultGenesisSpecPath = "../../chain/gssmr/genesis-spec.json"

	testCode := []byte("somecode")
	testHex := common.BytesToHex(testCode)
	testFile, err := ioutil.TempFile("", "testcode-*.wasm")
	require.NoError(t, err)
	defer os.Remove(testFile.Name())

	err = ioutil.WriteFile(testFile.Name(), testCode, 0777)
	require.NoError(t, err)

	out, err := createGenesisWithRuntime(testFile.Name())
	require.NoError(t, err)

	g := new(genesis.Genesis)
	err = json.Unmarshal([]byte(out), g)
	require.NoError(t, err)
	require.Equal(t, testHex, g.Genesis.Runtime["system"]["code"].(string))
}
