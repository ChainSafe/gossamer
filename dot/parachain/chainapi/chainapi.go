package chainapi

import (
	"context"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-chainapi"))

type ChainAPI struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	OverseerToSubSystem <-chan any
	SubSystemToOverseer chan<- any
}

func (c *ChainAPI) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
	go c.processMessages()
}

func (c *ChainAPI) Name() parachaintypes.SubSystemName {
	return parachaintypes.ChainAPI
}

func (c *ChainAPI) ProcessActiveLeavesUpdateSignal() {}

func (c *ChainAPI) ProcessBlockFinalizedSignal() {}

func (c *ChainAPI) Stop() {
	c.cancel()
	c.wg.Wait()
}

func (c *ChainAPI) processMessages() {
	for {
		select {
		case msg := <-c.OverseerToSubSystem:
			logger.Infof("received message %v", msg)

		case <-c.ctx.Done():
			if err := c.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v", err)
			}
			c.wg.Done()
			return
		}
	}
}

func Register(overseerChan chan<- any) (*ChainAPI, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	chainApiSubsystem := ChainAPI{
		ctx:                 ctx,
		cancel:              cancel,
		SubSystemToOverseer: overseerChan,
	}

	return &chainApiSubsystem, nil
}
