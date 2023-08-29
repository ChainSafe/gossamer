// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import "fmt"

type Overseer struct {
	subsystems        map[Subsystem]*Context
	subsystemMessages map[Subsystem]chan any
}

type ExamlpeSender struct {
	senderChan chan any
}

func (s *ExamlpeSender) SendMessage(msg any) error {
	s.senderChan <- msg
	return nil
}

func NewOverseer() *Overseer {
	return &Overseer{
		subsystems:        make(map[Subsystem]*Context),
		subsystemMessages: make(map[Subsystem]chan any),
	}
}

func (o *Overseer) RegisterSubsystem(subsystem Subsystem) {
	// TODO: determine best buffer size
	comChan := make(chan any, 5)
	o.subsystemMessages[subsystem] = comChan
	receiverChan := make(chan any, 5)
	o.subsystems[subsystem] = &Context{
		Sender:   &ExamlpeSender{senderChan: comChan},
		Receiver: receiverChan,
	}
}

func (o *Overseer) Start() {
	// start subsystems
	for subsystem, context := range o.subsystems {
		go func(sub Subsystem, ctx *Context) {
			err := sub.Run(ctx)
			if err != nil {
				// TODO: handle error
				fmt.Printf("error running subsystem %v: %v\n", sub, err)
			}
		}(subsystem, context)
	}

	// wait for messages from subsystems
	// TODO: this is a temporary solution, we will determine logic to handle different message types
	for subsystem, recChan := range o.subsystemMessages {
		go func(sub Subsystem, channel chan any) {
			fmt.Printf("overseer waiting for messages from %v\n", sub)

			for {
				select {
				case msg := <-channel:
					fmt.Printf("overseer received message from %v: %v\n", sub, msg)
				}
			}
		}(subsystem, recChan)
	}

	// TODO: add logic to start listening for Block Imported events and Finalisation events
}

func (o *Overseer) sendActiveLeavesUpdate(update *ActiveLeavesUpdate) {
	for _, context := range o.subsystems {
		context.Receiver <- update
	}
}
