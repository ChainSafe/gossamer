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
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
)

// ImportState imports the state in the given files to the database with the given path.
func ImportState(basepath, stateFP, headerFP string, firstSlot uint64) error {
	tr, err := newTrieFromPairs(stateFP)
	if err != nil {
		return err
	}

	header, err := newHeaderFromFile(headerFP)
	if err != nil {
		return err
	}

	log.Info("ImportState", "header", header)

	srv := state.NewService(basepath, log.LvlInfo)
	return srv.Import(header, tr, firstSlot)
}

func newTrieFromPairs(filename string) (*trie.Trie, error) {
	data, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	pairs := make([]interface{}, 0)
	err = json.Unmarshal(data, &pairs)
	if err != nil {
		return nil, err
	}

	entries := make(map[string]string)
	for _, pair := range pairs {
		pairArr := pair.([]interface{})
		if len(pairArr) != 2 {
			return nil, errors.New("state file contains invalid pair")
		}
		entries[pairArr[0].(string)] = pairArr[1].(string)
	}

	tr := trie.NewEmptyTrie()
	err = tr.LoadFromMap(entries)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

func newHeaderFromFile(filename string) (*types.Header, error) {
	data, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	jsonHeader := make(map[string]interface{})
	err = json.Unmarshal(data, &jsonHeader)
	if err != nil {
		return nil, err
	}

	hexNum, ok := jsonHeader["number"].(string)
	if !ok {
		return nil, errors.New("invalid number field in header JSON")
	}

	numBytes := common.MustHexToBytes(hexNum)
	num := big.NewInt(0).SetBytes(numBytes)

	parentHashStr, ok := jsonHeader["parentHash"].(string)
	if !ok {
		return nil, errors.New("invalid parentHash field in header JSON")
	}
	parentHash := common.MustHexToHash(parentHashStr)

	stateRootStr, ok := jsonHeader["stateRoot"].(string)
	if !ok {
		return nil, errors.New("invalid stateRoot field in header JSON")
	}
	stateRoot := common.MustHexToHash(stateRootStr)

	extrinsicsRootStr, ok := jsonHeader["extrinsicsRoot"].(string)
	if !ok {
		return nil, errors.New("invalid extrinsicsRoot field in header JSON")
	}
	extrinsicsRoot := common.MustHexToHash(extrinsicsRootStr)

	digestRaw, ok := jsonHeader["digest"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid digest field in header JSON")
	}
	logs := digestRaw["logs"].([]interface{})

	digest := types.Digest{}

	for _, log := range logs {
		digestBytes := common.MustHexToBytes(log.(string))
		r := &bytes.Buffer{}
		_, _ = r.Write(digestBytes)
		digestItem, err := types.DecodeDigestItem(r)
		if err != nil {
			return nil, err
		}

		digest = append(digest, digestItem)
	}

	header := &types.Header{
		ParentHash:     parentHash,
		Number:         num,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         digest,
	}

	return header, nil
}
