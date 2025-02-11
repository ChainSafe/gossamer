package bitfield_signing

import (
	"context"
	"fmt"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-signing"))

type signingTask struct {
	ctx      context.Context
	response <-chan parachaintypes.UncheckedSignedAvailabilityBitfield
}

type BitfieldSigning struct {
	subSystemToOverseer chan<- any
	keystore            keystore.Keystore
	tasks               map[common.Hash]*signingTask
}

func NewBitfieldSigning(overseerChan chan<- any, ks keystore.Keystore) *BitfieldSigning {
	return &BitfieldSigning{
		subSystemToOverseer: overseerChan,
		keystore:            ks,
	}
}

func (b *BitfieldSigning) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := b.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (b *BitfieldSigning) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := b.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			logger.Errorf("failed to process active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		err := b.ProcessBlockFinalizedSignal(msg)
		if err != nil {
			logger.Errorf("failed to process block finalized signal: %w", err)
		}
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (b *BitfieldSigning) Name() parachaintypes.SubSystemName {
	return parachaintypes.BitfieldSigning
}

func (b *BitfieldSigning) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	//TODO: implement in #4415
	panic("implement me")
}

func (b *BitfieldSigning) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	return nil
}

func (b *BitfieldSigning) Stop() {
	logger.Infof("Stopping BitfieldSigning subsystem")
}
