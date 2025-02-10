package bitfield_signing

import (
	"context"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// TODO: remove
var _ parachaintypes.Subsystem = (*BitfieldSigning)(nil)

type BitfieldSigning struct {
}

func (b BitfieldSigning) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	//TODO implement me
	panic("implement me")
}

func (b BitfieldSigning) Name() parachaintypes.SubSystemName {
	//TODO implement me
	panic("implement me")
}

func (b BitfieldSigning) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	//TODO implement me
	panic("implement me")
}

func (b BitfieldSigning) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	//TODO implement me
	panic("implement me")
}

func (b BitfieldSigning) Stop() {
	//TODO implement me
	panic("implement me")
}
