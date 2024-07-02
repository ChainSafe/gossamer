// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// This file contains helper functions for testing parachain.

package util

import (
	"context"
	"sync"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/database"
)

func NewTestOverseer() *TestOverseer {
	ctx := context.Background()

	return &TestOverseer{
		Ctx:                  ctx,
		Subsystems:           make(map[parachaintypes.Subsystem]chan any),
		SubsystemsToOverseer: make(chan any),
	}
}

type TestOverseer struct {
	Ctx context.Context
	Wg  sync.WaitGroup

	Subsystems           map[parachaintypes.Subsystem]chan any
	SubsystemsToOverseer chan any
}

func (to *TestOverseer) GetSubsystemToOverseerChannel() chan any {
	return to.SubsystemsToOverseer
}

func (to *TestOverseer) RegisterSubsystem(subsystem parachaintypes.Subsystem) chan any {
	overseerToSubSystem := make(chan any)
	to.Subsystems[subsystem] = overseerToSubSystem

	return overseerToSubSystem
}

func (to *TestOverseer) Start() error {
	// start subsystems
	for subsystem, overseerToSubSystem := range to.Subsystems {
		to.Wg.Add(1)
		go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
			sub.Run(to.Ctx, overseerToSubSystem, to.SubsystemsToOverseer)
			// logger.Infof("subsystem %v stopped", sub)
			to.Wg.Done()
		}(subsystem, overseerToSubSystem)
	}
	return nil
}

func (to *TestOverseer) Stop() error {
	return nil
}

func (to *TestOverseer) Broadcast(msg any) {
	for _, overseerToSubSystem := range to.Subsystems {
		overseerToSubSystem <- msg
	}
}

//== harness below, Not used now. need to change availability subsystem test code to use this code

// all subsystems should implement this interface to be able to run tests
type harnessConstructor interface {
	newHarnessTest(t *testing.T) *HarnessTest
}

type HarnessTest struct {
	overseer          *TestOverseer
	broadcastMessages []any
	broadcastIndex    int
	processes         []func(msg any)
	db                database.Database
}

// processes messgaes of other subsystems(return hardcoded responses)
// need to store function with hardcoded responses
func (h *HarnessTest) processMessages(t *testing.T) {
	processIndex := 0
	for {
		select {
		case msg := <-h.overseer.SubsystemsToOverseer:
			if h.processes != nil && processIndex < len(h.processes) {
				h.processes[processIndex](msg)
				processIndex++
			}
		case <-h.overseer.Ctx.Done():
			if err := h.overseer.Ctx.Err(); err != nil {
				t.Logf("ctx error: %v\n", err)
			}
			h.overseer.Wg.Done()
			return
		}
	}
}

// send a message from overseer to subsystem
func (h *HarnessTest) triggerBroadcast() {
	h.overseer.Broadcast(h.broadcastMessages[h.broadcastIndex])
	h.broadcastIndex++
}
