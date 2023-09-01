// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestSubsystem struct {
	name string
}

func (s *TestSubsystem) Run(ctx *Context) error {
	fmt.Printf("Run %s\n", s.name)
	// wait for leaf signal from overseer
	leaf := s.waitForLeaf(ctx)
	fmt.Printf("%s received leaf %s\n", s.name, leaf.Hash)

	// simulate work, send message to overseer
	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Printf("%v working on: %v\n", s.name, leaf)
			ctx.Sender.SendMessage(fmt.Sprintf("hello from %v", s.name))
		}
	}()

	return nil
}
func (s *TestSubsystem) waitForLeaf(context *Context) *ActivatedLeaf {
	for {
		select {
		case overseerSignal := <-context.Receiver:
			return overseerSignal.(*ActiveLeavesUpdate).Activated
		}
	}
}

func TestNewOverseer(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	overseer.Start()

	time.Sleep(500 * time.Millisecond)
	activedLeaf := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 3,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf})

	time.Sleep(5000 * time.Millisecond)

	overseer.Stop()
	time.Sleep(500 * time.Millisecond)
}
