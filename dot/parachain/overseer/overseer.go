// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"fmt"
	"github.com/ChainSafe/gossamer/internal/log"
	"sync"
)

const COMMS_BUFFER_SIZE = 5

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-overseer"))
)

type Overseer struct {
	wg     sync.WaitGroup
	doneCh chan struct{}
	stopCh chan struct{}

	subsystems        map[Subsystem]*Context
	subsystemMessages map[Subsystem]chan any
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
		subsystemMessages: make(map[Subsystem]chan any),
	}
}

func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	// TODO: determine best buffer size
	comChan := make(chan any, COMMS_BUFFER_SIZE)
	o.subsystemMessages[subsystem] = comChan
	receiverChan := make(chan any, COMMS_BUFFER_SIZE)
	o.subsystems[subsystem] = &Context{
		Sender:   &ExampleSender{senderChan: comChan},
		Receiver: receiverChan,
	}
}

func (o *Overseer) Start() {
	// start subsystems
	for subsystem, context := range o.subsystems {
		o.wg.Add(1)
		go func(sub Subsystem, ctx *Context, wg sync.WaitGroup) {
			defer wg.Done()
			select {
			case <-o.stopCh:
				return
			}
			err := sub.Run(ctx)
			if err != nil {
				logger.Errorf("running subsystem %v failed: %v", sub, err)
			}
		}(subsystem, context, o.wg)
	}

	// wait for messages from subsystems
	// TODO: this is a temporary solution, we will determine logic to handle different message types
	for subsystem, recChan := range o.subsystemMessages {
		o.wg.Add(1)
		go func(sub Subsystem, channel chan any, wg sync.WaitGroup) {
			defer wg.Done()
			fmt.Printf("overseer waiting for messages from %v\n", sub)

			for {
				select {
				case <-o.stopCh:
					return
				case msg := <-channel:
					fmt.Printf("overseer received message from %v: %v\n", sub, msg)
				}
			}
		}(subsystem, recChan, o.wg)
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

func (o *Overseer) sendActiveLeavesUpdate(update *ActiveLeavesUpdate) {
	for _, context := range o.subsystems {
		context.Receiver <- update
	}
}
