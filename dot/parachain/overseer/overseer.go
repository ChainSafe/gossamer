// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
)

const CommsBufferSize = 5

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-overseer"))
)

type Overseer struct {
	messageListener Sender
	wg              sync.WaitGroup
	doneCh          chan struct{}
	//stopCh          chan struct{}
	errChan chan error

	subsystems map[Subsystem]*context
}

type exampleSender struct {
}

func (s *exampleSender) SendMessage(msg any) error {
	fmt.Printf("exampleSender sending message: %v\n", msg)
	return nil
}

func NewOverseer() *Overseer {
	return &Overseer{
		messageListener: &exampleSender{},
		doneCh:          make(chan struct{}),
		//stopCh:          make(chan struct{}),
		errChan:    make(chan error),
		subsystems: make(map[Subsystem]*context),
	}
}

// RegisterSubsystem registers a subsystem with the overseer,
//
//		Add context to subsystem map, which includes: Sender implementation,
//		Receiver channel.  The context will be pass to subsystem's Run method
//		Subsystem will use that context to send messages to overseer, and to recieve messages from overseer (
//		via receiver channel), and to signal when it is done (overseer closes the receiver channel)
//	  the subsystem will signal overseer when it's done stopping by done channel
//		Overseer implements SendMessage interface for Subsystem to communitate to overseer
//		Overseer returns channel to subsystem for messages from overseer to subsystem,
//
// and when that channel is closed it uses that to confirm subsystem is done
func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	receiverChan := make(chan any, CommsBufferSize)
	o.subsystems[subsystem] = &context{
		Sender:   &exampleSender{},
		Receiver: receiverChan,
		wg:       &o.wg,
		//stopCh:   o.stopCh,
	}
}

func (o *Overseer) Start() (errChan chan error, err error) {
	// start subsystems
	for subsystem, cntxt := range o.subsystems {
		o.wg.Add(1)
		go func(sub Subsystem, ctx *context) {
			err := sub.Run(ctx)
			if err != nil {
				logger.Errorf("running subsystem %v failed: %v", sub, err)
			}
		}(subsystem, cntxt)
	}

	// TODO: add logic to start listening for Block Imported events and Finalisation events
	return o.errChan, nil
}

func (o *Overseer) Stop() error {
	if o.doneCh == nil {
		return nil
	}
	//close(o.stopCh)

	for subsystem, _ := range o.subsystems {
		close(o.subsystems[subsystem].Receiver)
	}

	timeout := time.Millisecond
	waitTimeout(&o.wg, timeout)

	// close the errorChan to unblock any listeners on the errChan
	close(o.errChan)
	//o.stopCh = nil
	return nil
}

func (o *Overseer) sendActiveLeavesUpdate(update *ActiveLeavesUpdate, subsystem Subsystem) {
	o.subsystems[subsystem].Receiver <- update
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
