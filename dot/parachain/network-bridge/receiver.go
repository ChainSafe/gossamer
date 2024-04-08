package networkbridge

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type NetworkBridgeReceiver struct{}

func (nbs *NetworkBridgeReceiver) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) {
}

func (nbs *NetworkBridgeReceiver) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeReceiver
}

func (nbs *NetworkBridgeReceiver) ProcessActiveLeavesUpdateSignal() {}

func (nbs *NetworkBridgeReceiver) ProcessBlockFinalizedSignal() {}

func (nbs *NetworkBridgeReceiver) Stop() {}
