package disputescoordinator

import (
	"context"
	"fmt"

	disputescoordinatormessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-disputes-coordinator"))

type DisputesCoordinator struct {
	SubSystemToOverseer chan<- any
}

func New(overseerChan chan<- any) *DisputesCoordinator {
	return &DisputesCoordinator{
		SubSystemToOverseer: overseerChan,
	}
}

func (*DisputesCoordinator) Name() parachaintypes.SubSystemName {
	return parachaintypes.DisputesCoordinator
}

func (dc *DisputesCoordinator) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			if err := dc.processMessage(msg); err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			return
		}
	}
}

func (dc *DisputesCoordinator) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		if err := dc.ProcessActiveLeavesUpdateSignal(msg); err != nil {
			return fmt.Errorf("process active leaves update signal: %w", err)
		}
		return nil
	case parachaintypes.BlockFinalizedSignal:
		return dc.ProcessBlockFinalizedSignal(msg)
	case disputescoordinatormessages.QueryCandidateVotes:
		msg.Response <- []disputescoordinatormessages.CandidateVotesResponse{}
		return nil
	case disputescoordinatormessages.GetRecentDisputes:
		msg.Response <- []disputescoordinatormessages.RecentDispute{}
		return nil
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
}

func (cd *DisputesCoordinator) ProcessActiveLeavesUpdateSignal(
	update parachaintypes.ActiveLeavesUpdateSignal,
) error {
	panic("not implemented yet")
}

func (cd *DisputesCoordinator) ProcessBlockFinalizedSignal(
	parachaintypes.BlockFinalizedSignal,
) error {
	// Nothing to do here
	return nil
}

func (cd *DisputesCoordinator) Stop() {}
