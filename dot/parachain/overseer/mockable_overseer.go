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

	// expected actions for overseer messages we receive from the subsystem.
	// need to return false if the message is unexpected
	actions []func(msg any) bool
}

func NewMockableOverseer(t *testing.T) *MockableOverseer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MockableOverseer{
		t:                    t,
		ctx:                  ctx,
		cancel:               cancel,
		SubsystemsToOverseer: make(chan any),
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
		sub.Run(m.ctx, overseerToSubSystem, m.SubsystemsToOverseer)
	}(m.subSystem, m.overseerToSubsystem)

	go m.processMessages()
	return nil
}

func (m *MockableOverseer) Stop() {
	m.cancel()
	close(m.overseerToSubsystem)
}

// ReceiveMessage method is to receive overseer messages in a subsystem which we are testing
func (m *MockableOverseer) ReceiveMessage(msg any) {
	m.overseerToSubsystem <- msg
}

// ExpectActions method is to set expected actions for overseer messages we receive from the subsystem.
// actions are expected in the order they are set.
// all the functions in the arguments should return false if the message is unexpected.
func (m *MockableOverseer) ExpectActions(fns ...func(msg any) bool) {
	m.actions = append(m.actions, fns...)
}

func (m *MockableOverseer) processMessages() { //nolint:unused
	actionIndex := 0
	for {
		select {
		case msg := <-m.SubsystemsToOverseer:
			if msg == nil {
				continue
			}

			if actionIndex < len(m.actions) {
				action := m.actions[actionIndex]
				ok := action(msg)
				if !ok {
					m.t.Errorf("unexpected message: %T", msg)
					return
				}

				actionIndex = actionIndex + 1
			}
		case <-m.ctx.Done():
			if err := m.ctx.Err(); err != nil {
				m.t.Logf("ctx error: %v\n", err)
			}
			return
		}
	}
}
