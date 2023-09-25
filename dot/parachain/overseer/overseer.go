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
	ctx        context.Context
	cancel     context.CancelFunc
	errChan    chan error // channel for overseer to send errors to service that started it
	subsystems map[Subsystem]*overseerContext
	wg         sync.WaitGroup
}

func NewOverseer() *Overseer {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	return &Overseer{
		ctx:        ctx,
		cancel:     cancel,
		errChan:    make(chan error),
		subsystems: make(map[Subsystem]*overseerContext),
	}
}

// RegisterSubsystem registers a subsystem with the overseer,
//
// Add overseerContext to subsystem map, which includes: context, Sender implementation,
// Receiver channel.  The overseerContext will be pass to subsystem's Run method
// Subsystem will use that overseerContext to send messages to overseer, and to receive messages from overseer (
// via receiver channel), and context to signal when overseer has canceled
func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	o.subsystems[subsystem] = &overseerContext{
		ctx:          o.ctx,
		ToOverseer:   make(chan any),
		FromOverseer: make(chan any),
	}
}

func (o *Overseer) Start() error {
	// start subsystems
	for subsystem, cntxt := range o.subsystems {
		o.wg.Add(1)
		go func(sub Subsystem, ctx *overseerContext) {
			err := sub.Run(ctx)
			if err != nil {
				logger.Errorf("running subsystem %v failed: %v", sub, err)
			}
			logger.Infof("subsystem %v stopped", sub)
			o.wg.Done()
		}(subsystem, cntxt)
	}

	// TODO: add logic to start listening for Block Imported events and Finalisation events
	return nil
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
	o.subsystems[subsystem].FromOverseer <- update
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
