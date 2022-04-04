// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils/retry"
	"github.com/ChainSafe/gossamer/tests/utils/rpc"
)

func waitForNode(ctx context.Context, rpcPort string) (err error) {
	const retryWait = time.Second
	err = retry.UntilNoError(ctx, retryWait, func() (err error) {
		const checkNodeStartedTimeout = time.Second
		checkNodeCtx, checkNodeCancel := context.WithTimeout(ctx, checkNodeStartedTimeout)
		err = checkNodeStarted(checkNodeCtx, "http://localhost:"+rpcPort)
		checkNodeCancel()
		return err
	})

	if err != nil {
		return fmt.Errorf("node did not start: %w", err)
	}

	return nil
}

var errNodeNotExpectingPeers = errors.New("node should expect to have peers")

// checkNodeStarted check if gossamer node is started
func checkNodeStarted(ctx context.Context, gossamerHost string) error {
	health, err := rpc.GetHealth(ctx, gossamerHost)
	if err != nil {
		return fmt.Errorf("cannot get health: %w", err)
	}

	if !health.ShouldHavePeers {
		return errNodeNotExpectingPeers
	}

	return nil
}
