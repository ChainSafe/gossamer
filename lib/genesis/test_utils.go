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
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"
)

const testProtocolID = "/gossamer/test/0"

var testBootnodes = []string{
	"/dns4/p2p.cc3-0.kusama.network/tcp/30100/p2p/QmeCit3Nif4VfNqrEJsdYHZGcKzRCnZvGxg6hha1iNj4mk",
	"/dns4/p2p.cc3-1.kusama.network/tcp/30100/p2p/QmchDJtEGiEWf7Ag58HNoTg9jSGzxkSZ23VgmF6xiLKKsZ",
}

var testEndpoints = []interface{}{}
var testEndpoint1 = []interface{}{"wss://telemetry.polkadot.io/submit/", float64(1)}
var testProperties = map[string]interface{}{"ss58Format": float64(0), "tokenDecimals": float64(10), "tokenSymbol": "DOT"}

var testForkBlocks = []string{"fork1", "forkBlock2"}

var testBadBlocks = []string{"badBlock1", "badBlock2"}

// TestGenesis instance of Genesis struct for testing
var TestGenesis = &Genesis{
	Name:               "gossamer",
	ID:                 "gossamer",
	Bootnodes:          testBootnodes,
	TelemetryEndpoints: append(testEndpoints, testEndpoint1),
	ProtocolID:         testProtocolID,
	Properties:         testProperties,
	ForkBlocks:         testForkBlocks,
	BadBlocks:          testBadBlocks,
}

// TestFieldsHR instance of human-readable Fields struct for testing, use with TestGenesis
var TestFieldsHR = Fields{
	Raw: map[string]map[string]string{},
	Runtime: map[string]map[string]interface{}{
		"System": {
			"code": "mocktestcode",
		},
	},
}

// TestFieldsRaw instance of raw Fields struct for testing use with TestGenesis
var TestFieldsRaw = Fields{
	Raw: map[string]map[string]string{
		"top": {
			"0x3a636f6465": "mocktestcode",
			common.BytesToHex(common.UpgradedToDualRefKey): "0x01",
		},
	},
}

// CreateTestGenesisJSONFile utility to create mock test genesis JSON file
func CreateTestGenesisJSONFile(asRaw bool) (string, error) {
	// Create temp file
	file, err := ioutil.TempFile("", "genesis-test")
	if err != nil {
		return "", err
	}

	tGen := &Genesis{
		Name:       "test",
		ID:         "",
		Bootnodes:  nil,
		ProtocolID: "",
		Genesis:    Fields{},
	}

	if asRaw {
		tGen.Genesis = Fields{
			Raw: map[string]map[string]string{},
			Runtime: map[string]map[string]interface{}{
				"System": {
					"code": "mocktestcode",
				},
			},
		}
	} else {
		tGen.Genesis = TestFieldsHR
	}

	bz, err := json.Marshal(tGen)
	if err != nil {
		return "", nil
	}
	// Write to temp file
	_, err = file.Write(bz)
	if err != nil {
		return "", nil
	}

	return file.Name(), nil
}

// NewTestGenesisWithTrieAndHeader generates genesis, genesis trie and genesis header
func NewTestGenesisWithTrieAndHeaderVdt(t *testing.T) (*Genesis, *trie.Trie, *types.HeaderVdt) {
	gen, err := NewGenesisFromJSONRaw("../../chain/gssmr/genesis.json")
	if err != nil {
		gen, err = NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
		require.NoError(t, err)
	}

	genTrie, err := NewTrieFromGenesis(gen)
	require.NoError(t, err)

	genesisHeader, err := types.NewHeader(common.NewHash([]byte{0}), genTrie.MustHash(), trie.EmptyHash, big.NewInt(0), types.NewDigestVdt())
	require.NoError(t, err)
	return gen, genTrie, genesisHeader
}
