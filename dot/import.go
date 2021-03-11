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
	"io/ioutil"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"

	log "github.com/ChainSafe/log15"
)

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
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	if err != nil {
		return nil, err
	}

	pairs := rpcPairs["result"].([]interface{})

	entries := make(map[string]string)
	for _, pair := range pairs {
		pairArr := pair.([]interface{})
		entries[pairArr[0].(string)] = pairArr[1].(string)
	}

	tr := trie.NewEmptyTrie()
	err = tr.LoadFromMap(entries)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

// {"digest":{"logs":["0x0642414245b501017f0000007caae90f00000000f665037728a7dcb3eafb2d34c2464e4dc724bf7ddf0528fb843ecb7d89e0c73bbe64f95b574f8e0f52ef33996deb1ae8890dcd35789c7dd6146a0e14efef8900e1663c7495029d56901394624a9fc866186b8dfa13d24044c4b20bdeca29780b","0x054241424501013018f04729d567caa1509516ac34559e7dfaf77e6a12533b81abc22ad4b5c431831289593dc72ec8d604f3d3bef9422d5cd04076af66f17e4f73581d0f3e9e8a"]},"extrinsicsRoot":"0xa6078bb4f8900a576b55a1d63955132a5cd0aabd2911ace65a79b7071776f9a6","number":"0x421aa1","parentHash":"0x1999508ae6d5a96365fe8f397c00059d43e53ae96c609ac539c666a06a011317","stateRoot":"0x3fd8e00fad457fc2ed6b06239d805a262fe5d79797739311461069d91f556923"}
func newHeaderFromFile(filename string) (*types.Header, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	jsonHeader := make(map[string]interface{})
	err = json.Unmarshal(data, &jsonHeader)
	if err != nil {
		return nil, err
	}

	hexNum := jsonHeader["number"].(string)
	numBytes := common.MustHexToBytes(hexNum)
	num := big.NewInt(0).SetBytes(numBytes)

	parentHashStr := jsonHeader["parentHash"].(string)
	parentHash := common.MustHexToHash(parentHashStr)

	stateRootStr := jsonHeader["stateRoot"].(string)
	stateRoot := common.MustHexToHash(stateRootStr)

	extrinsicsRootStr := jsonHeader["extrinsicsRoot"].(string)
	extrinsicsRoot := common.MustHexToHash(extrinsicsRootStr)

	digestRaw := jsonHeader["digest"].(map[string]interface{})
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
