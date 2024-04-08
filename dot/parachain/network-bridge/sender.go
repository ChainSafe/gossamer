package networkbridge

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type NetworkBridgeSender struct{}

func (nbs *NetworkBridgeSender) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
}

func (nbs *NetworkBridgeSender) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeSender
}

func (nbs *NetworkBridgeSender) ProcessActiveLeavesUpdateSignal() {}

func (nbs *NetworkBridgeSender) ProcessBlockFinalizedSignal() {}

func (nbs *NetworkBridgeSender) Stop() {}
