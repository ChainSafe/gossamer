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
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	"io/ioutil"
	"math/big"
	"path/filepath"
)

// NewGenesisFromJSON parses a JSON formatted genesis file
func NewGenesisFromJSON(file string) (*Genesis, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(Genesis)
	err = json.Unmarshal(data, g)
	return g, err
}

func NewGenesisFromJSONHR(file string) (*Genesis, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(Genesis)

	err = json.Unmarshal(data, g)
	fmt.Printf("raw top %v\n", g.Genesis.Raw["top"])
	top := g.Genesis.Runtime
	fmt.Printf("raw %v\n", top)

	res := printMap(top)
	for k, v := range res {
		fmt.Printf(    "k: %v vT: %v\n", k, len(fmt.Sprint(v)))
	}
	fmt.Printf("grandpa %v\n", res["[grandpa authorities]"])
	fmt.Printf("sudo key %v\n", res["[sudo key]"])
	return g, err
}

type KeyValue struct {
	key []string
	value string
	valueLen *big.Int
}

func printMap(m map[string]map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		kv := new(KeyValue)
		fmt.Printf("k: %v\n", k)
		kv.key = append(kv.key, k)
		printMapInterface(v, kv)
		fmt.Printf("kvLen %v\n", len(kv.key))
		fmt.Printf("kv %s\n", kv.key)
		// todo check and encode key
		res[fmt.Sprint(kv.key)] = kv.value
		fmt.Printf("value len %v\n", kv.valueLen)
	}
	return res
}

func printMapInterface(m map[string]interface{}, kv *KeyValue) {
	for k, v := range m {
		fmt.Printf("\tk %v\n", k)
		kv.key = append(kv.key, k)
		switch v2 := v.(type) {
		//case map[string]interface{}:
		//	fmt.Printf("Got as live one %v\n", reflect.TypeOf(v2))
		//	printMapInterface(v2, kv)
		case []interface{}:
			fmt.Printf("Got an array!!!! %v\n", len(v2))
			kv.valueLen = big.NewInt(int64(len(v2)))
			printArrayInterface(v2, kv)
		case string:
			fmt.Printf("\t\t sVl %v\n", len(v2))
			kv.value = v2
		}
	}
}

func printArrayInterface(a []interface{}, kv *KeyValue) {
	for _, v := range a {
		switch v2 := v.(type) {
		case []interface{}:
			printArrayInterface(v2, kv)
		case string:
			// todo check to confirm it's an address
			fmt.Printf("\t\t aSl %v, val: %v\n", len(v2), v2)
			tba := crypto.PublicAddressToByteArray(common.Address(v2))
			fmt.Printf("\t\t publictobyte bytes %x len: %v\n", tba, len(tba))
			kv.value = kv.value + fmt.Sprintf("%x", tba)
		case float64:
			//01 00 00 00 00 00 00 00
			encVal, err := scale.Encode(uint64(v2))
			if err != nil {
				fmt.Errorf("error encoding number")
			}
			fmt.Printf("\t\t float %v\n", v2)
			fmt.Printf("\t\t float as enc byte %x\n", encVal)
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
		}
	}
}

// NewTrieFromGenesis creates a new trie from the raw genesis data
func NewTrieFromGenesis(g *Genesis) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()

	r := g.GenesisFields().Raw["top"]

	err := t.Load(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create trie from genesis: %s", err)
	}

	return t, nil
}

// NewGenesisBlockFromTrie creates a genesis block from the provided trie
func NewGenesisBlockFromTrie(t *trie.Trie) (*types.Header, error) {

	// create state root from trie hash
	stateRoot, err := t.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to create state root from trie hash: %s", err)
	}

	// create genesis block header
	header, err := types.NewHeader(
		common.NewHash([]byte{0}), // parentHash
		big.NewInt(0),             // number
		stateRoot,                 // stateRoot
		trie.EmptyHash,            // extrinsicsRoot
		[][]byte{},                // digest
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block header: %s", err)
	}

	return header, nil
}
