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

package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

func TestNewGenesisRawFromJSON(t *testing.T) {
	// Create temp file
	file, err := ioutil.TempFile("", "genesis-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	testRaw := map[string]map[string]string{}
	testRaw["top"] = map[string]string{"0x3a636f6465": "0x0102"}

	expected := TestGenesis
	expected.Genesis = Fields{Raw: testRaw}

	// Grab json encoded bytes
	bz, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	// Write to temp file
	_, err = file.Write(bz)
	if err != nil {
		t.Fatal(err)
	}

	genesis, err := NewGenesisFromJSONRaw(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, expected, genesis)
}

func TestNewGenesisFromJSON(t *testing.T) {
	var expectedGenesis = &Genesis{}

	expRaw := make(map[string]map[string]string)
	expRaw["top"] = make(map[string]string)
	expRaw["top"]["0x3a636f6465"] = "0xfoo"
	expRaw["top"]["0x3a6772616e6470615f617574686f726974696573"] = "0x010834602b88f60513f1c805d87ef52896934baf6a662bc37414dbdbf69356b1a6910000000000000000"                                                                                                                                                                                             // raw grandpa authorities
	expRaw["top"]["0x1cb6f36e027abb2091cfb5110ab5087f5e0621c4869aa60c02be9adcc98a0d1d"] = "0x08d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000"                                                                                                                                                                       // raw babe authorities
	expRaw["top"]["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"] = "0x0000000000000000000000007aeb9049000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" // raw system account
	expRaw["top"][common.BytesToHex(common.UpgradedToDualRefKey)] = "0x01"

	expectedGenesis.Genesis = Fields{
		Raw: expRaw,
	}

	// Create temp file
	file, err := ioutil.TempFile("", "genesis_hr-test")
	require.NoError(t, err)

	defer os.Remove(file.Name())

	// create human readable test genesis
	testGenesis := &Genesis{}
	hrData := make(map[string]map[string]interface{})
	hrData["System"] = map[string]interface{}{"code": "0xfoo"} // system code entry
	hrData["Babe"] = make(map[string]interface{})
	hrData["Babe"]["Authorities"] = []interface{}{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1} // babe authority data
	hrData["Grandpa"] = make(map[string]interface{})
	hrData["Grandpa"]["authorities"] = []interface{}{"5DFNv4Txc4b88qHqQ6GG4D646QcT4fN3jjS2G3r1PyZkfDut", 0} // grandpa authority data
	hrData["Balances"] = make(map[string]interface{})
	hrData["Balances"]["balances"] = []interface{}{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1234234234} // balances
	testGenesis.Genesis = Fields{
		Runtime: hrData,
	}

	// Grab json encoded bytes
	bz, err := json.Marshal(testGenesis)
	require.NoError(t, err)
	// Write to temp file
	_, err = file.Write(bz)
	require.NoError(t, err)

	// create genesis based on file just created, this will fill Raw field of genesis
	testGenesisProcessed, err := NewGenesisFromJSON(file.Name(), 0)
	require.NoError(t, err)

	require.Equal(t, expectedGenesis.Genesis.Raw, testGenesisProcessed.Genesis.Raw)
}

func TestFormatKey(t *testing.T) {
	kv := &keyValue{
		key: []string{"Babe", "Authorities"},
	}

	out, err := formatKey(kv)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("0x%x", runtime.BABEAuthoritiesKey()), out)
}

func TestNewTrieFromGenesis(t *testing.T) {
	var rawGenesis = &Genesis{}
	raw := make(map[string]map[string]string)
	raw["top"] = make(map[string]string)
	raw["top"]["0x3a636f6465"] = "0x0102" // raw :code
	rawGenesis.Genesis = Fields{
		Raw: raw,
	}

	expTrie := trie.NewEmptyTrie()
	expTrie.Put([]byte(`:code`), []byte{1, 2})

	trie, err := NewTrieFromGenesis(rawGenesis)
	require.NoError(t, err)

	require.Equal(t, expTrie, trie)
}
