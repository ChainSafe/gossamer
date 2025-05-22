// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"context"
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-statement-distribution"))

type BlockState interface {
	GetHeader(hash common.Hash) (header *types.Header, err error)
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

type StatementDistribution struct {
	blockState          BlockState
	SubSystemToOverseer chan<- any
	state               *v2State
}

func New(overseerChan chan<- any, ks keystore.Keystore, blockState parachainutil.BlockState) *StatementDistribution {
	return &StatementDistribution{
		SubSystemToOverseer: overseerChan,
		blockState:          blockState,
		state:               newV2State(ks, parachainutil.NewBackingImplicitView(blockState, nil)),
	}
}

// Run just receives the ctx and a channel from the overseer to subsystem
func (s *StatementDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	// Inside the method Run, we spawn a goroutine to handle network incoming requests
	// TODO: https://github.com/ChainSafe/gossamer/issues/4285
	responderCh := make(chan any, 1)
	go taskResponder(responderCh)

	// Timer for reputation aggregator trigger
	reputationDelay := time.NewTicker(parachainutil.ReputationChangeInterval) // Adjust the duration as needed
	defer reputationDelay.Stop()

	for {
		message := s.awaitMessageFrom(overseerToSubSystem, responderCh, reputationDelay.C)

		switch innerMessage := message.(type) {
		case *reputationChangeMessage:
			logger.Info("Reputation change triggered.")
		case *overseerMessage:
			shouldStop, err := s.handleSubsystemMessage(innerMessage.inner)
			if err != nil {
				logger.Errorf("handling subsystem message: %s", err.Error())
			}

			if shouldStop {
				logger.Warn("handling subsystem message: should stop statement distribution")
				break
			}
		default:
			logger.Warn("Unhandled message type: " + fmt.Sprintf("%v", innerMessage))
		}
	}
}

func (s *StatementDistribution) handleSubsystemMessage(overseerMessage any) (bool, error) {
	switch message := overseerMessage.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		if message.Activated != nil {
			if err := s.handleActiveLeavesUpdate(message.Activated); err != nil {
				return false, fmt.Errorf("handling active leaves update: %w", err)
			}
		}
		s.handleDeactivatedLeaves(message.Deactivated)

	case parachaintypes.Conclude:
		return true, nil
	}

	return false, nil
}

func (s *StatementDistribution) sendPeerMessageForRelayParent(pid string, rp common.Hash) {
	panic("unimplemented")
}

func (s *StatementDistribution) newLeafFragmentChainUpdates(rp common.Hash) {
	panic("unimplemented")
}

// TODO: https://github.com/ChainSafe/gossamer/issues/4285
func taskResponder(responderCh chan any) {}
