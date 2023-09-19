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

	for {
		select {
		case overseerSignal, ok := <-ctx.Receiver:
			stop := make(chan struct{})
			fmt.Printf("Ok: %v, sig %v\n", ok, overseerSignal)
			if !ok {
				close(stop)
			}
			// simulate work, send message to overseer
			go func(stop chan struct{}) {
				counter := 0
				for {
					select {
					case <-stop:
						fmt.Printf("overseer stopping %v\n", s.name)
						return
					default:
					}

					counter++
					time.Sleep(time.Second)
					fmt.Printf("%v working on: %v\n", s.name, overseerSignal)
					ctx.Sender.SendMessage(fmt.Sprintf("hello from %v, count: %v", s.name, counter))
				}
			}(stop)
		}
	}
	return nil
}

func TestStart2SubsytemsActivate1(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	errChan, err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)

	// let subsystems run for a bit
	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}

func TestStart2SubsytemsActivate2Different(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	errChan, err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

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

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}

func TestStart2SubsytemsActivate2Same(t *testing.T) {
	overseer := NewOverseer()
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	errChan, err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf := &ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)
	overseer.sendActiveLeavesUpdate(&ActiveLeavesUpdate{Activated: activedLeaf}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(5000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}
