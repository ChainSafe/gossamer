// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"context"
	"sync"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type MockableOverseer struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SubsystemsToOverseer chan any
	overseerToSubsystem  chan any
	subSystem            parachaintypes.Subsystem

	msgToAction map[any]func(msg any)
}

func NewMockableOverseer() *MockableOverseer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MockableOverseer{
		ctx:                  ctx,
		cancel:               cancel,
		wg:                   sync.WaitGroup{},
		SubsystemsToOverseer: make(chan any),
		msgToAction:          make(map[any]func(msg any)),
	}
}

func (m *MockableOverseer) GetSubsystemToOverseerChannel() chan any {
	return m.SubsystemsToOverseer
}

func (m *MockableOverseer) RegisterSubsystem(subsystem parachaintypes.Subsystem) chan any {
	OverseerToSubSystem := make(chan any)
	m.overseerToSubsystem = OverseerToSubSystem
	m.subSystem = subsystem
	return OverseerToSubSystem
}

func (m *MockableOverseer) Start() error {
	m.wg.Add(2)

	go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
		sub.Run(m.ctx, overseerToSubSystem, m.SubsystemsToOverseer)
	}(m.subSystem, m.overseerToSubsystem)

	go m.processMessages()

	return nil
}

func (m *MockableOverseer) Stop() {
}

func (m *MockableOverseer) ReceiveMessage(msg any) {
	m.overseerToSubsystem <- msg
}

func (m *MockableOverseer) MockMessageAction(msg any, fn func(msg any)) {
	m.msgToAction[msg] = fn
}

func (m *MockableOverseer) processMessages() {
	for {
		msg := <-m.SubsystemsToOverseer
		m.msgToAction[msg](msg)
	}
}
