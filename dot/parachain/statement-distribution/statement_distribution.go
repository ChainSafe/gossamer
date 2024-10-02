package statementdistribution

import (
	"context"

	"github.com/ChainSafe/gossamer/internal/log"

	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statement-distribution"))

type StatementDistribution struct {
}

func (s StatementDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			err := s.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
		}
	}
}

func (s StatementDistribution) processMessage(msg any) error {

	switch msg := msg.(type) {
	case statementedistributionmessages.Backed:
		// TODO #4171
	case statementedistributionmessages.Share:
		// TODO #4170
	// case statementedistributionmessages.NetworkBridgeUpdate
	// TODO #4172 this above case would need to wait until network bridge receiver side is merged
	case parachaintypes.ActiveLeavesUpdateSignal:
		return s.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		return s.ProcessBlockFinalizedSignal(msg)

	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil
}

func (s StatementDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.StatementDistribution
}

func (s StatementDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO #4173
	return nil
}

func (s StatementDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (s StatementDistribution) Stop() {}
