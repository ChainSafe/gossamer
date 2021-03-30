// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package stress

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

var (
	maxRetries  = 32
	testTimeout = time.Minute * 3
	logger      = log.New("pkg", "tests/stress")
)

// compareChainHeads calls getChainHead for each node in the array
// it returns a map of chainHead hashes to node key names, and an error if the hashes don't all match
func compareChainHeads(t *testing.T, nodes []*utils.Node) (map[common.Hash][]string, error) {
	hashes := make(map[common.Hash][]string)
	for _, node := range nodes {
		header := utils.GetChainHead(t, node)
		logger.Info("got header from node", "hash", header.Hash(), "node", node.Key)
		hashes[header.Hash()] = append(hashes[header.Hash()], node.Key)
	}

	var err error
	if len(hashes) != 1 {
		err = errChainHeadMismatch
	}

	return hashes, err
}

// compareChainHeadsWithRetry calls compareChainHeads, retrying up to maxRetries times if it errors.
func compareChainHeadsWithRetry(t *testing.T, nodes []*utils.Node) error {
	var hashes map[common.Hash][]string
	var err error

	for i := 0; i < maxRetries; i++ {
		hashes, err = compareChainHeads(t, nodes)
		if err == nil {
			break
		}

		time.Sleep(time.Second)
	}

	if err != nil {
		err = fmt.Errorf("%w: hashes=%v", err, hashes)
	}

	return err
}

// compareBlocksByNumber calls getBlockByNumber for each node in the array
// it returns a map of block hashes to node key names, and an error if the hashes don't all match
func compareBlocksByNumber(t *testing.T, nodes []*utils.Node, num string) (map[common.Hash][]string, error) {
	hashes := make(map[common.Hash][]string)
	var errs []error
	var mapMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node *utils.Node) {
			logger.Debug("calling chain_getBlockHash", "node", node.Idx)
			hash, err := utils.GetBlockHash(t, node, num)
			mapMu.Lock()
			defer func() {
				mapMu.Unlock()
				wg.Done()
			}()
			if err != nil {
				errs = append(errs, err)
				return
			}
			logger.Debug("got hash from node", "hash", hash, "node", node.Key)

			hashes[hash] = append(hashes[hash], node.Key)
		}(node)
	}

	wg.Wait()

	var err error
	if len(errs) != 0 {
		err = fmt.Errorf("%v", errs)
	}

	if len(hashes) == 0 {
		err = errNoBlockAtNumber
	}

	if len(hashes) > 1 {
		err = errBlocksAtNumberMismatch
	}

	return hashes, err
}

// compareBlocksByNumberWithRetry calls compareChainHeads, retrying up to maxRetries times if it errors.
func compareBlocksByNumberWithRetry(t *testing.T, nodes []*utils.Node, num string) (map[common.Hash][]string, error) {
	var hashes map[common.Hash][]string
	var err error

	timeout := time.After(30 * time.Second)
doneBlockProduction:
	for {
		time.Sleep(time.Second)
		select {
		case <-timeout:
			break doneBlockProduction
		default:
			hashes, err = compareBlocksByNumber(t, nodes, num)
			if err == nil {
				break doneBlockProduction
			}
		}
	}

	if err != nil {
		err = fmt.Errorf("%w: hashes=%v", err, hashes)
	}
	return hashes, err
}

// compareFinalizedHeads calls getFinalizedHeadByRound for each node in the array
// it returns a map of finalizedHead hashes to node key names, and an error if the hashes don't all match
func compareFinalizedHeads(t *testing.T, nodes []*utils.Node) (map[common.Hash][]string, error) {
	hashes := make(map[common.Hash][]string)
	for _, node := range nodes {
		hash := utils.GetFinalizedHead(t, node)
		logger.Info("got finalized head from node", "hash", hash, "node", node.Key)
		hashes[hash] = append(hashes[hash], node.Key)
	}

	var err error
	if len(hashes) == 0 {
		err = errNoFinalizedBlock
	}

	if len(hashes) > 1 {
		err = errFinalizedBlockMismatch
	}

	return hashes, err
}

// compareFinalizedHeadsByRound calls getFinalizedHeadByRound for each node in the array
// it returns a map of finalizedHead hashes to node key names, and an error if the hashes don't all match
func compareFinalizedHeadsByRound(t *testing.T, nodes []*utils.Node, round uint64) (map[common.Hash][]string, error) {
	hashes := make(map[common.Hash][]string)
	for _, node := range nodes {
		hash, err := utils.GetFinalizedHeadByRound(t, node, round)
		if err != nil {
			return nil, err
		}

		logger.Info("got finalized head from node", "hash", hash, "node", node.Key, "round", round)
		hashes[hash] = append(hashes[hash], node.Key)
	}

	var err error
	if len(hashes) == 0 {
		err = errNoFinalizedBlock
	}

	if len(hashes) > 1 {
		err = errFinalizedBlockMismatch
	}

	return hashes, err
}

// compareFinalizedHeadsWithRetry calls compareFinalizedHeadsByRound, retrying up to maxRetries times if it errors.
// it returns the finalized hash if it succeeds
func compareFinalizedHeadsWithRetry(t *testing.T, nodes []*utils.Node, round uint64) (common.Hash, error) {
	var hashes map[common.Hash][]string
	var err error

	for i := 0; i < maxRetries; i++ {
		hashes, err = compareFinalizedHeadsByRound(t, nodes, round)
		if err == nil {
			break
		}

		if err == errFinalizedBlockMismatch {
			return common.Hash{}, fmt.Errorf("%w: round=%d hashes=%v", err, round, hashes)
		}

		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return common.Hash{}, fmt.Errorf("%w: round=%d hashes=%v", err, round, hashes)
	}

	for h := range hashes {
		return h, nil
	}

	return common.Hash{}, nil
}

func getPendingExtrinsics(t *testing.T, node *utils.Node) []string { //nolint
	respBody, err := utils.PostRPC(utils.AuthorPendingExtrinsics, utils.NewEndpoint(node.RPCPort), "[]")
	require.NoError(t, err)

	exts := new(modules.PendingExtrinsicsResponse)
	err = utils.DecodeRPC(t, respBody, exts)
	require.NoError(t, err)

	return *exts
}
