// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"context"
	"errors"
	"sync"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type MockableOverseer struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SubsystemsToOverseer chan any
	overseerToSubsystem  chan any
	subSystem            parachaintypes.Subsystem

	// this is going to be limited to only one testcase
	msgToAction map[any]func(msg any)
}

func NewMockableOverseer() *MockableOverseer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MockableOverseer{
		ctx:    ctx,
		cancel: cancel,
		wg:     sync.WaitGroup{},

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
	ctx := context.Background()
	go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
		logger.Info("starting subsystem...")
		sub.Run(ctx, overseerToSubSystem, m.SubsystemsToOverseer)
		logger.Info("subsystem stopped.")
	}(m.subSystem, m.overseerToSubsystem)

	return nil
}

func (m *MockableOverseer) Stop() {
}

func (m *MockableOverseer) ReceiveMessage(msg any) {
	m.overseerToSubsystem <- msg
}

type InputOutput struct {
	InputMessage  any
	OutputMessage any
}

func (m *MockableOverseer) MockMessageAction(msg any, fn func(msg any)) {
	m.msgToAction[msg] = fn
}

func test(msg any) {
	newMessage := msg.(parachaintypes.ProspectiveParachainsMessageIntroduceCandidate)
	newMessage.Ch <- errors.New("error")
}

func (m *MockableOverseer) processMessages() {
	for {
		select {
		case msg := <-m.SubsystemsToOverseer:
			m.msgToAction[msg](msg)
		}
	}
}

func preSetup(t *testing.T) {
	t.Helper()

	overseer := NewMockableOverseer()

	backing := New(overseer.SubsystemsToOverseer)
	// candidateBacking.BlockState = nil // use mock block state

	backing.ctx = overseer.ctx
	backing.cancel = overseer.cancel
	backing.OverseerToSubSystem = overseer.RegisterSubsystem(backing)

	overseer.Start()
}

// func TestBacking(t *testing.T) {
// 	go preSetup(t)
// 	time.Sleep(2 * time.Second)
// 	println("Test backing")

// 	time.Sleep(10 * time.Second)
// }
