// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"

	"github.com/stretchr/testify/require"
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "tests/stress"))
)

// compareChainHeads calls getChainHead for each node in the array
// it returns a map of chainHead hashes to node key names, and an error if the hashes don't all match
func compareChainHeads(ctx context.Context, nodes node.Nodes,
	getChainHeadTimeout time.Duration) (hashes map[common.Hash][]string, err error) {
	hashes = make(map[common.Hash][]string)
	for _, node := range nodes {
		getChainHeadCtx, cancel := context.WithTimeout(ctx, getChainHeadTimeout)
		header, err := rpc.GetChainHead(getChainHeadCtx, node.RPCPort())
		cancel()
		if err != nil {
			return nil, fmt.Errorf("cannot get chain head for node %s: %w", node, err)
		}

		logger.Infof("got header with hash %s from node %s", header.Hash(), node)
		hashes[header.Hash()] = append(hashes[header.Hash()], node.Key())
	}

	if len(hashes) != 1 {
		err = errChainHeadMismatch
	}

	return hashes, err
}

// compareChainHeadsWithRetry calls compareChainHeads,
// retrying until the context gets canceled.
func compareChainHeadsWithRetry(ctx context.Context, nodes node.Nodes,
	getChainHeadTimeout time.Duration) error {
	for {
		hashes, err := compareChainHeads(ctx, nodes, getChainHeadTimeout)
		if err == nil {
			return nil
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("%w: hashes=%v", err, hashes) // last error
		}
	}
}

var errBlockHashNotOne = errors.New("expected 1 block hash")

// compareBlocksByNumber calls getBlockByNumber for each node in the array
// it returns a map of block hashes to node key names, and an error if the hashes don't all match
func compareBlocksByNumber(ctx context.Context, nodes node.Nodes,
	num string) (nodeKeys []string, err error) {
	blockHashes := make(map[common.Hash]struct{}, 1)
	for _, n := range nodes {
		const retryWait = time.Second
		err := retry.UntilOK(ctx, retryWait, func() (ok bool, err error) {
			hash, err := rpc.GetBlockHash(ctx, n.RPCPort(), num)
			if err != nil {
				const blockDoesNotExistString = "cannot find node with number greater than highest in blocktree"
				if strings.Contains(err.Error(), blockDoesNotExistString) {
					return false, nil // retry after retryWait has elapsed.
				}
				return false, err // stop retrying
			}

			blockHashes[hash] = struct{}{}
			nodeKeys = append(nodeKeys, n.Key())
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("for node %s and block number %s: %w", n, num, err)
		}
	}

	if len(blockHashes) != 1 {
		return nil, fmt.Errorf("%w: but got %d block hashes for block number %s",
			errBlockHashNotOne, len(blockHashes), num)
	}

	return nodeKeys, nil
}

// compareFinalizedHeads calls getFinalizedHeadByRound for each node in the array
// it returns a map of finalisedHead hashes to node key names, and an error if the hashes don't all match
func compareFinalizedHeads(ctx context.Context, t *testing.T, nodes node.Nodes,
	getFinalizedHeadTimeout time.Duration) (hashes map[common.Hash][]string, err error) {
	hashes = make(map[common.Hash][]string)
	for _, node := range nodes {
		getFinalizedHeadCtx, cancel := context.WithTimeout(ctx, getFinalizedHeadTimeout)
		hash, err := rpc.GetFinalizedHead(getFinalizedHeadCtx, node.RPCPort())
		cancel()
		require.NoError(t, err)

		logger.Infof("got finalised head with hash %s from node %s", hash, node)
		hashes[hash] = append(hashes[hash], node.Key())
	}

	if len(hashes) == 0 {
		err = errNoFinalizedBlock
	}

	if len(hashes) > 1 {
		err = errFinalizedBlockMismatch
	}

	return hashes, err
}

// compareFinalizedHeadsByRound calls getFinalizedHeadByRound for each node in the array
// it returns a map of finalisedHead hashes to node key names, and an error if the hashes don't all match
func compareFinalizedHeadsByRound(ctx context.Context, nodes node.Nodes,
	round uint64) (err error) {
	hashes := make(map[common.Hash][]string)
	for _, node := range nodes {
		const getFinalizedHeadByRoundTimeout = time.Second
		getFinalizedHeadByRoundCtx, cancel := context.WithTimeout(ctx, getFinalizedHeadByRoundTimeout)
		hash, err := rpc.GetFinalizedHeadByRound(getFinalizedHeadByRoundCtx, node.RPCPort(), round)
		cancel()

		if err != nil {
			return fmt.Errorf("cannot get finalized head for round %d: %w", round, err)
		}

		logger.Infof("got finalised head with hash %s from node %s at round %d", hash, node, round)
		hashes[hash] = append(hashes[hash], node.Key())
	}

	switch len(hashes) {
	case 0:
		return errNoFinalizedBlock
	case 1:
		return nil
	default:
		return errFinalizedBlockMismatch
	}
}
