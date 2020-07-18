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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/OneOfOne/xxhash"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"reflect"
	"strings"
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

	res := buildRawMap(top)
	for k, v := range res {
		fmt.Printf(    "k: %v vT: %v\n", k, len(fmt.Sprint(v)))
	}
	fmt.Printf("grandpa %v\n", res["0x3a6772616e6470615f617574686f726974696573"])
	codeS := fmt.Sprintf("%v",res["0x3a636f6465"])

	fmt.Printf("code %v\n", codeS[:30])
	fmt.Printf("2xHash Babe Authorities %v\n", res["0x886726f904d8372fdabb7707870c2fad"])
	fmt.Printf("sudo key %v\n", res["0x50a63a871aced22e88ee6466fe5aa5d9"])
	return g, err
}

type KeyValue struct {
	key []string
	value string
	valueLen *big.Int
}

func buildRawMap(m map[string]map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		kv := new(KeyValue)
		fmt.Printf("k: %v\n", k)
		kv.key = append(kv.key, k)
		buildRawMapInterface(v, kv)
		fmt.Printf("kvLen %v\n", len(kv.key))
		fmt.Printf("kv %s\n", kv.key)
		// todo check and encode key
		key := formatKey(kv.key)
		fmt.Printf("Formatted Key %v\n", key)
		value, err := formatValue(kv)
		if err != nil {
			// todo determine how to handle error
		}
		res[key] = value
		fmt.Printf("value len %v\n", kv.valueLen)
	}
	return res
}

func buildRawMapInterface(m map[string]interface{}, kv *KeyValue) {
	for k, v := range m {
		fmt.Printf("\tk %v\n", k)
		kv.key = append(kv.key, k)
		switch v2 := v.(type) {
		//case map[string]interface{}:
		//	fmt.Printf("Got as live one %v\n", reflect.TypeOf(v2))
		//	buildRawMapInterface(v2, kv)
		case []interface{}:
			fmt.Printf("Got an array!!!! %v\n", len(v2))
			kv.valueLen = big.NewInt(int64(len(v2)))
			buildRawArrayInterface(v2, kv)
		case string:
			fmt.Printf("\t\t sVl %v\n", len(v2))
			kv.value = v2
		}
	}
}

func buildRawArrayInterface(a []interface{}, kv *KeyValue) {
	for _, v := range a {
		switch v2 := v.(type) {
		case []interface{}:
			buildRawArrayInterface(v2, kv)
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

func formatKey(key []string) string {
	switch true {
	case equal([]string{"grandpa", "authorities"}, key):
		kb := []byte(`:grandpa_authorities`)
		return common.BytesToHex(kb)
	case equal([]string{"system", "code"}, key):
		kb := []byte(`:code`)
		return common.BytesToHex(kb)
	default:
		var fKey string
		for _, v := range key {
			fKey = fKey + v + " "
		}
		fKey = strings.Trim(fKey, " ")
		fKey = strings.Title(fKey)
		fmt.Printf("fKey:%v:\n", fKey)
		fmt.Printf("fKey byte %v\n", []byte(fKey))
		kb := TwoxHash([]byte(fKey))
		return common.BytesToHex(kb)
	}
}

func formatValue(kv *KeyValue) (string, error) {
	switch true {
	case reflect.DeepEqual([]string{"grandpa", "authorities"}, kv.key):
		if kv.valueLen != nil {
			lenEnc, err := scale.Encode(kv.valueLen)
			if err != nil {
				return "", err
			}
			// prepend 01 to grandpa_authorities values
			return fmt.Sprintf("0x01%x%v", lenEnc, kv.value), nil
		}
		return "", fmt.Errorf("error formatting value for grandpa authorities")
	case reflect.DeepEqual([]string{"system", "code"}, kv.key):
		return kv.value, nil
	default:
		if kv.valueLen != nil {
			lenEnc, err := scale.Encode(kv.valueLen)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("0x%x%v", lenEnc, kv.value), nil
		}
		return fmt.Sprintf("0x%x", kv.value), nil
	}
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
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

func TwoxHash(msg []byte) []byte {
	//memory := []byte(`System Number`)  // 0x8cb577756012d928f17362e0741f9f2c
	//logger.Trace("[ext_twox_128] hashing...", "value", fmt.Sprintf("%s", memory[:]))

	// compute xxHash64 twice with seeds 0 and 1 applied on given byte array
	h0 := xxhash.NewS64(0) // create xxHash with 0 seed
	_, err := h0.Write(msg[0 : len(msg)])
	if err != nil {
		//logger.Error("[ext_twox_128]", "error", err)
		return nil
	}
	res0 := h0.Sum64()
	hash0 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash0, res0)

	h1 := xxhash.NewS64(1) // create xxHash with 1 seed
	_, err = h1.Write(msg[0 : len(msg)])
	if err != nil {
		//logger.Error("[ext_twox_128]", "error", err)
		return nil
	}
	res1 := h1.Sum64()
	hash1 := make([]byte, 8)
	binary.LittleEndian.PutUint64(hash1, res1)

	//concatenated result
	both := append(hash0, hash1...)
	fmt.Printf("both: %x\n", both)
	return both
}