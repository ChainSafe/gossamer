// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-overseer"))
)

type Overseer struct {
	ctx            context.Context
	cancel         context.CancelFunc
	errChan        chan error // channel for overseer to send errors to service that started it
	FromSubsystems chan any
	subsystems     map[Subsystem]chan any // map[Subsystem]OverseerToSubSystem channel
	wg             sync.WaitGroup
}

func NewOverseer() *Overseer {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	return &Overseer{
		ctx:            ctx,
		cancel:         cancel,
		errChan:        make(chan error),
		FromSubsystems: make(chan any),
		subsystems:     make(map[Subsystem]chan any),
	}
}

// RegisterSubsystem registers a subsystem with the overseer,
// Add OverseerToSubSystem channel to subsystem, which will be passed to subsystem's Run method.
func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	o.subsystems[subsystem] = make(chan any)
}

func (o *Overseer) Start() error {
	// start subsystems
	for subsystem, fromOverseerToSubSystem := range o.subsystems {
		o.wg.Add(1)
		go func(sub Subsystem, fromOverseer chan any) {
			err := sub.Run(o.ctx, o.FromSubsystems, fromOverseer)
			if err != nil {
				logger.Errorf("running subsystem %v failed: %v", sub, err)
			}
			logger.Infof("subsystem %v stopped", sub)
			o.wg.Done()
		}(subsystem, fromOverseerToSubSystem)
	}

	go o.processMessages()

	// TODO: add logic to start listening for Block Imported events and Finalisation events
	return nil
}

func (o *Overseer) processMessages() {
	for {
		select {
		case msg := <-o.FromSubsystems:
			switch msg.(type) {
			default:
				logger.Error("unknown message type")
			}
		case <-o.ctx.Done():
			if err := o.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			logger.Info("overseer stopping")
			return
		}
	}
}

func (o *Overseer) Stop() error {
	o.cancel()

	// close the errorChan to unblock any listeners on the errChan
	close(o.errChan)

	// wait for subsystems to stop
	// TODO: determine reasonable timeout duration for production, currently this is just for testing
	timedOut := waitTimeout(&o.wg, 500*time.Millisecond)
	fmt.Printf("timedOut: %v\n", timedOut)

	return nil
}

// sendActiveLeavesUpdate sends an ActiveLeavesUpdate to the subsystem
func (o *Overseer) sendActiveLeavesUpdate(update ActiveLeavesUpdate, subsystem Subsystem) {
	o.subsystems[subsystem] <- update
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
