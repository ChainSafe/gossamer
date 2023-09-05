// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

const CommsBufferSize = 5

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-overseer"))
)

type Overseer struct {
	wg     sync.WaitGroup
	doneCh chan struct{}
	stopCh chan struct{}

	subsystems        map[Subsystem]*Context
	subsystemMessages map[Subsystem]<-chan any
	overseerChannel   chan any
}

type ExampleSender struct {
	senderChan chan any
}

func (s *ExampleSender) SendMessage(msg any) error {
	s.senderChan <- msg
	return nil
}

func NewOverseer() *Overseer {
	return &Overseer{
		doneCh:            make(chan struct{}),
		stopCh:            make(chan struct{}),
		subsystems:        make(map[Subsystem]*Context),
		subsystemMessages: make(map[Subsystem]<-chan any),
		overseerChannel:   make(chan any),
	}
}

func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	o.subsystemMessages[subsystem] = o.overseerChannel
	receiverChan := make(chan any, CommsBufferSize)
	o.subsystems[subsystem] = &Context{
		Sender:   &ExampleSender{senderChan: o.overseerChannel},
		Receiver: receiverChan,
		wg:       &o.wg,
		stopCh:   o.stopCh,
	}
}

func (o *Overseer) Start() {
	// start subsystems
	for subsystem, context := range o.subsystems {
		o.wg.Add(1)
		go func(sub Subsystem, ctx *Context) {
			err := sub.Run(ctx)
			if err != nil {
				logger.Errorf("running subsystem %v failed: %v", sub, err)
			}
		}(subsystem, context)
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
}

func (o *Overseer) Stop() {
	if o.doneCh == nil {
		return
	}
	close(o.stopCh)
	o.wg.Wait()
	o.stopCh = nil
}

func (o *Overseer) sendActiveLeavesUpdate(update *ActiveLeavesUpdate, subsystem Subsystem) {
	o.subsystems[subsystem].Receiver <- update
	//for _, context := range o.subsystems {
	//	context.Receiver <- update
	//}
}
