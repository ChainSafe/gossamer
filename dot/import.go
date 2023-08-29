// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package dot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/ChainSafe/gossamer/internal/log"
)

// ImportState imports the state in the given files to the database with the given path.
func ImportState(basepath, stateFP, headerFP string, firstSlot uint64, stateVersion trie.Version) error {
	tr, err := newTrieFromPairs(stateFP, stateVersion)
	if err != nil {
		return err
	}

	header, err := newHeaderFromFile(headerFP)
	if err != nil {
		return err
	}

	logger.Infof("ImportState with header: %v", header)

	config := state.Config{
		Path:     basepath,
		LogLevel: log.Info,
	}
	srv := state.NewService(config)
	return srv.Import(header, tr, firstSlot)
}

func newTrieFromPairs(filename string, stateVersion trie.Version) (*trie.Trie, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
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

	tr, err := trie.LoadFromMap(entries, stateVersion)
	if err != nil {
		return nil, err
	}

	return &tr, nil
}

func newHeaderFromFile(filename string) (*types.Header, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
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

	num, err := common.HexToUint(hexNum)
	if err != nil {
		return nil, fmt.Errorf("cannot convert number field: %w", err)
	}

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

	digest := types.NewDigest()

	for _, log := range logs {
		digestBytes := common.MustHexToBytes(log.(string))
		var digestItem = types.NewDigestItem()
		err := scale.Unmarshal(digestBytes, &digestItem)
		if err != nil {
			return nil, err
		}

		digestItemVal, err := digestItem.Value()
		if err != nil {
			return nil, fmt.Errorf("getting digest item value: %w", err)
		}
		err = digest.Add(digestItemVal)
		if err != nil {
			return nil, err
		}
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
