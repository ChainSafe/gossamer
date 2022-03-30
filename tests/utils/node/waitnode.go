package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/tests/utils/rpc"
)

func waitForNode(ctx context.Context, rpcPort string) (err error) {
	tries := 0
	const checkNodeStartedTimeout = time.Second
	const retryWait = time.Second
	for ctx.Err() == nil {
		tries++

		checkNodeCtx, checkNodeCancel := context.WithTimeout(ctx, checkNodeStartedTimeout)

		err = checkNodeStarted(checkNodeCtx, "http://localhost:"+rpcPort)
		checkNodeCancel()
		if err == nil {
			return nil
		}

		retryWaitCtx, retryWaitCancel := context.WithTimeout(ctx, retryWait)
		<-retryWaitCtx.Done()
		retryWaitCancel()
	}

	totalTryTime := time.Duration(tries) * checkNodeStartedTimeout
	tryWord := "try"
	if tries > 1 {
		tryWord = "tries"
	}
	return fmt.Errorf("node did not start after %d %s during %s: %w",
		tries, tryWord, totalTryTime, err)
}

var errNodeNotExpectingPeers = errors.New("node shoult expect to have peers")

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
