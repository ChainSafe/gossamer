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
		}
	}

}

func (s StatementDistribution) processMessage(msg any) error {

	switch msg := msg.(type) {
	case statementedistributionmessages.Backed:
		// todo
	case statementedistributionmessages.Share:
		// todo
	// case statementedistributionmessages.NetworkBridgeUpdate
	// TODO this above case would need to wait
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
	// todo
	return nil
}

func (s StatementDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (s StatementDistribution) Stop() {}
