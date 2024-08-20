// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	types "github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type TestSubsystem struct {
	name             string
	finalizedCounter atomic.Int32
	importedCounter  atomic.Int32
}

func (s *TestSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("%s ctx error: %v\n", s.name, err)
			}
			fmt.Printf("%s overseer stopping\n", s.name)
			return
		case overseerSignal := <-overseerToSubSystem:
			fmt.Printf("%s received from overseer %v\n", s.name, overseerSignal)
			incrementCounters(overseerSignal, &s.finalizedCounter, &s.importedCounter)
		}
	}
}

func (s *TestSubsystem) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	fmt.Printf("%s ProcessActiveLeavesUpdateSignal\n", s.name)
	return nil
}

func (s *TestSubsystem) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	fmt.Printf("%s ProcessActiveLeavesUpdateSignal\n", s.name)
	return nil
}

func (s *TestSubsystem) String() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Stop() {}

func TestHandleBlockEvents(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{
		name:             "subSystem1",
		finalizedCounter: atomic.Int32{},
		importedCounter:  atomic.Int32{},
	}

	subSystem2 := &TestSubsystem{
		name:             "subSystem2",
		finalizedCounter: atomic.Int32{},
		importedCounter:  atomic.Int32{},
	}

	overseer.RegisterSubsystem(subSystem1)
	overseer.RegisterSubsystem(subSystem2)

	err := overseer.Start()
	require.NoError(t, err)
	finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{}

	// let subsystems run for a bit
	time.Sleep(4000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	require.Equal(t, int32(1), subSystem1.finalizedCounter.Load())
	require.Equal(t, int32(1), subSystem1.importedCounter.Load())
	require.Equal(t, int32(1), subSystem2.finalizedCounter.Load())
	require.Equal(t, int32(1), subSystem2.importedCounter.Load())
}

func incrementCounters(msg any, finalizedCounter *atomic.Int32, importedCounter *atomic.Int32) {
	if msg == nil {
		return
	}

	switch msg.(type) {
	case parachaintypes.BlockFinalizedSignal:
		finalizedCounter.Add(1)
	case parachaintypes.ActiveLeavesUpdateSignal:
		importedCounter.Add(1)
	}
}
