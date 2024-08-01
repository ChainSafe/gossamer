// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type MockableOverseer struct {
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc

	SubsystemsToOverseer chan any
	overseerToSubsystem  chan any
	subSystem            parachaintypes.Subsystem

	expectedMessagesWithAction map[any]func(msg any)
}

func NewMockableOverseer(t *testing.T) *MockableOverseer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MockableOverseer{
		t:                          t,
		ctx:                        ctx,
		cancel:                     cancel,
		SubsystemsToOverseer:       make(chan any),
		expectedMessagesWithAction: make(map[any]func(msg any)),
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
	go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
		sub.Run(m.ctx, m.cancel, overseerToSubSystem, m.SubsystemsToOverseer)
	}(m.subSystem, m.overseerToSubsystem)

	go m.processMessages()
	return nil
}

func (m *MockableOverseer) Stop() {
	m.cancel()
}

func (m *MockableOverseer) ReceiveMessage(msg any) {
	m.overseerToSubsystem <- msg
}

func (m *MockableOverseer) ExpectMessageWithAction(msg any, fn func(msg any)) {
	m.expectedMessagesWithAction[msg] = fn
}

func (m *MockableOverseer) processMessages() {
	for {
		select {
		case msg := <-m.SubsystemsToOverseer:
			if msg == nil {
				continue
			}
			action, ok := m.expectedMessagesWithAction[msg]
			if !ok {
				m.t.Errorf("unexpected message: %v", msg)
				continue
			}

			action(msg)
		case <-m.ctx.Done():
			if err := m.ctx.Err(); err != nil {
				m.t.Logf("ctx error: %v\n", err)
			}
			return
		}
	}
}
