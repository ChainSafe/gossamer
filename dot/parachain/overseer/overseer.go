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
	wg      sync.WaitGroup
	doneCh  chan struct{}
	stopCh  chan struct{}
	errChan chan error

	subsystems        map[Subsystem]*context
	subsystemMessages map[Subsystem]<-chan any
	overseerChannel   chan any
}

type exampleSender struct {
	senderChan chan any
}

func (s *exampleSender) SendMessage(msg any) error {
	s.senderChan <- msg
	return nil
}

func NewOverseer() *Overseer {
	return &Overseer{
		doneCh:            make(chan struct{}),
		stopCh:            make(chan struct{}),
		errChan:           make(chan error),
		subsystems:        make(map[Subsystem]*context),
		subsystemMessages: make(map[Subsystem]<-chan any),
		overseerChannel:   make(chan any),
	}
}

func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	o.subsystemMessages[subsystem] = o.overseerChannel
	receiverChan := make(chan any, CommsBufferSize)
	o.subsystems[subsystem] = &context{
		Sender:   &exampleSender{senderChan: o.overseerChannel},
		Receiver: receiverChan,
		wg:       &o.wg,
		stopCh:   o.stopCh,
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

	// wait for messages from subsystems
	// TODO: this is a temporary solution, we will determine logic to handle different message types
	for subsystem, recChan := range o.subsystemMessages {
		go func(sub Subsystem, channel <-chan any) {
			fmt.Printf("overseer waiting for messages from %v\n", sub)
			for { //nolint:gosimple
				select {
				case msg := <-channel:
					fmt.Printf("overseer received message from %v: %v\n", sub, msg)
				}
			}
		}(subsystem, recChan)
	}

	// TODO: add logic to start listening for Block Imported events and Finalisation events
	return o.errChan, nil
}

func (o *Overseer) Stop() error {
	if o.doneCh == nil {
		return nil
	}
	close(o.stopCh)
	timeout := 5 * time.Second

	waitTimeout(&o.wg, timeout)

	// close the errorChan to unblock any listeners on the errChan
	close(o.errChan)
	o.stopCh = nil
	return nil
}

func (o *Overseer) sendActiveLeavesUpdate(update *ActiveLeavesUpdate, subsystem Subsystem) {
	o.subsystems[subsystem].Receiver <- update
	//for _, context := range o.subsystems {
	//	context.Receiver <- update
	//}
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
