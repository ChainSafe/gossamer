// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestSubsystem struct {
	name string
}

func (s *TestSubsystem) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error {
	fmt.Printf("%s run\n", s.name)
	counter := 0
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("%s ctx error: %v\n", s.name, err)
			}
			fmt.Printf("%s overseer stopping\n", s.name)
			return nil
		case overseerSignal := <-OverseerToSubSystem:
			fmt.Printf("%s received from overseer %v\n", s.name, overseerSignal)
		default:
			// simulate work, and sending messages to overseer
			r := rand.Intn(1000)
			time.Sleep(time.Duration(r) * time.Millisecond)
			SubSystemToOverseer <- fmt.Sprintf("hello from %v, i: %d", s.name, counter)
			counter++
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

	err := overseer.Start()
	require.NoError(t, err)

	done := make(chan struct{})
	// listen for errors from overseer
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	time.Sleep(1000 * time.Millisecond)
	activedLeaf := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)

	// let subsystems run for a bit
	time.Sleep(4000 * time.Millisecond)

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

	err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf1 := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	activedLeaf2 := ActivatedLeaf{
		Hash:   [32]byte{2},
		Number: 2,
	}
	time.Sleep(250 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdate{Activated: activedLeaf1}, subSystem1)
	time.Sleep(400 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdate{Activated: activedLeaf2}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(3000 * time.Millisecond)

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

	err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	time.Sleep(300 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdate{Activated: activedLeaf}, subSystem1)
	time.Sleep(400 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdate{Activated: activedLeaf}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(2000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}
