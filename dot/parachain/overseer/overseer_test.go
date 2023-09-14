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

func (s *TestSubsystem) Run(ctx *context) error {
	fmt.Printf("Run %s\n", s.name)
	// wait for leaf signal from overseer
	leaf := s.waitForLeaf(ctx)
	fmt.Printf("%s received leaf %v\n", s.name, leaf)

	// simulate work, send message to overseer
	go func() {
		defer ctx.wg.Done()
		counter := 0
		for {
			select {
			case <-ctx.stopCh:
				fmt.Printf("overseer stopping %v\n", s.name)
				return
			default:
			}

			counter++
			time.Sleep(time.Second)
			fmt.Printf("%v working on: %v\n", s.name, leaf)
			ctx.Sender.SendMessage(fmt.Sprintf("hello from %v, count: %v", s.name, counter))
		}
	}()

	return nil
}
func (s *TestSubsystem) waitForLeaf(context *context) *ActivatedLeaf {
	for {
		select {
		case overseerSignal := <-context.Receiver:
			return overseerSignal.(*ActiveLeavesUpdate).Activated
		case <-context.stopCh: //listen for stop signal here in case we need to stop before we get a leaf
			fmt.Printf("overseer stopping %v\n", s.name)
			return nil
		}
	}
}

func TestStart2SubsytemsActivate1(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	overseer.Start()

	activedLeaf := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)

	// let subsystems run for a bit
	time.Sleep(5000 * time.Millisecond)

	overseer.Stop()
	time.Sleep(time.Second)
	fmt.Printf("overseer stopped\n")
}

func TestStart2SubsytemsActivate2Different(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	overseer.Start()

	activedLeaf1 := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	activedLeaf2 := &ActivatedLeaf{
		Hash:   [32]byte{2},
		Number: 2,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf1}, subSystem1)
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf2}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(5000 * time.Millisecond)

	overseer.Stop()
	time.Sleep(time.Second)
	fmt.Printf("overseer stopped\n")
}

func TestStart2SubsytemsActivate2Same(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	overseer.Start()

	activedLeaf := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(5000 * time.Millisecond)

	overseer.Stop()
	time.Sleep(time.Second)
	fmt.Printf("overseer stopped\n")
}
