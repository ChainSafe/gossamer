// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package stress

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/tests/utils/node"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"

	"github.com/stretchr/testify/require"
)

var (
	maxRetries  = 32
	testTimeout = time.Minute * 3
	logger      = log.NewFromGlobal(log.AddContext("pkg", "tests/stress"))
)

// compareChainHeads calls getChainHead for each node in the array
// it returns a map of chainHead hashes to node key names, and an error if the hashes don't all match
func compareChainHeads(ctx context.Context, nodes node.Nodes,
	getChainHeadTimeout time.Duration) (hashes map[common.Hash][]string, err error) {
	hashes = make(map[common.Hash][]string)
	for _, node := range nodes {
		getChainHeadCtx, cancel := context.WithTimeout(ctx, getChainHeadTimeout)
		header, err := rpc.GetChainHead(getChainHeadCtx, node.GetRPCPort())
		cancel()
		if err != nil {
			return nil, fmt.Errorf("cannot get chain head for node %s: %w", node, err)
		}

		logger.Infof("got header with hash %s from node %s", header.Hash(), node)
		hashes[header.Hash()] = append(hashes[header.Hash()], node.GetKey())
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
	var hashes map[common.Hash][]string
	var err error

	for i := 0; i < maxRetries; i++ {
		hashes, err = compareChainHeads(ctx, nodes, getChainHeadTimeout)
		if err == nil {
			break
		}

		timer := time.NewTimer(time.Second)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return err // last error
		}
	}

	if err != nil {
		err = fmt.Errorf("%w: hashes=%v", err, hashes)
	}

	return err
}

// compareBlocksByNumber calls getBlockByNumber for each node in the array
// it returns a map of block hashes to node key names, and an error if the hashes don't all match
func compareBlocksByNumber(ctx context.Context, t *testing.T, nodes node.Nodes,
	num string) (hashToKeys map[common.Hash][]string) {
	type resultContainer struct {
		hash    common.Hash
		nodeKey string
		err     error
	}
	results := make(chan resultContainer)

	for _, n := range nodes {
		go func(node node.Node) {
			result := resultContainer{
				nodeKey: node.GetKey(),
			}

			for { // retry until context gets canceled
				result.hash, result.err = rpc.GetBlockHash(ctx, node.GetRPCPort(), num)

				if err := ctx.Err(); err != nil {
					result.err = err
					break
				}

				if result.err == nil {
					break
				}
			}

			results <- result
		}(n)
	}

	var err error
	hashToKeys = make(map[common.Hash][]string, len(nodes))
	for range nodes {
		result := <-results
		if err != nil {
			continue // one failed, we don't care anymore
		}

		if result.err != nil {
			err = result.err
			continue
		}

		hashToKeys[result.hash] = append(hashToKeys[result.hash], result.nodeKey)
	}

	require.NoError(t, err)
	require.Lenf(t, hashToKeys, 1,
		"expected 1 block found for number %s but got %d block(s)",
		num, len(hashToKeys))

	return hashToKeys
}

// compareFinalizedHeads calls getFinalizedHeadByRound for each node in the array
// it returns a map of finalisedHead hashes to node key names, and an error if the hashes don't all match
func compareFinalizedHeads(ctx context.Context, t *testing.T, nodes node.Nodes,
	getFinalizedHeadTimeout time.Duration) (hashes map[common.Hash][]string, err error) {
	hashes = make(map[common.Hash][]string)
	for _, node := range nodes {
		getFinalizedHeadCtx, cancel := context.WithTimeout(ctx, getFinalizedHeadTimeout)
		hash, err := rpc.GetFinalizedHead(getFinalizedHeadCtx, node.GetRPCPort())
		cancel()
		require.NoError(t, err)

		logger.Infof("got finalised head with hash %s from node %s", hash, node)
		hashes[hash] = append(hashes[hash], node.GetKey())
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
	round uint64, getFinalizedHeadByRoundTimeout time.Duration) (
	hashes map[common.Hash][]string, err error) {
	hashes = make(map[common.Hash][]string)
	for _, node := range nodes {
		getFinalizedHeadByRoundCtx, cancel := context.WithTimeout(ctx, getFinalizedHeadByRoundTimeout)
		hash, err := rpc.GetFinalizedHeadByRound(getFinalizedHeadByRoundCtx, node.GetRPCPort(), round)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("cannot get finalized head for round %d: %w", round, err)
		}

		logger.Infof("got finalised head with hash %s from node %s at round %d", hash, node, round)
		hashes[hash] = append(hashes[hash], node.GetKey())
	}

	if len(hashes) == 0 {
		err = errNoFinalizedBlock
	}

	if len(hashes) > 1 {
		err = errFinalizedBlockMismatch
	}

	return hashes, err
}

// compareFinalizedHeadsWithRetry calls compareFinalizedHeadsByRound, retrying up to maxRetries times if it errors.
// it returns the finalised hash if it succeeds
func compareFinalizedHeadsWithRetry(ctx context.Context, nodes node.Nodes, round uint64,
	getFinalizedHeadByRoundTimeout time.Duration) (hash common.Hash, err error) {
	var hashes map[common.Hash][]string

	for i := 0; i < maxRetries; i++ {
		hashes, err = compareFinalizedHeadsByRound(ctx, nodes, round, getFinalizedHeadByRoundTimeout)
		if err == nil {
			break
		}

		if errors.Is(err, errFinalizedBlockMismatch) {
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

func getPendingExtrinsics(ctx context.Context, t *testing.T, node node.Node) []string {
	endpoint := rpc.NewEndpoint(node.GetRPCPort())
	const method = "author_pendingExtrinsics"
	const params = "[]"
	respBody, err := rpc.Post(ctx, endpoint, method, params)
	require.NoError(t, err)

	exts := new(modules.PendingExtrinsicsResponse)
	err = rpc.Decode(respBody, exts)
	require.NoError(t, err)

	return *exts
}
