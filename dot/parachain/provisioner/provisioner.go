// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package provisioner

import (
	"context"
	"time"

	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-provisioner"))

// inherentPreProposeTimeout is the duration to wait before inherent data is ready
const inherentPreProposeTimeout = 2 * time.Second

func New() *Provisioner {
	return &Provisioner{
		perRelayParent: make(map[common.Hash]perRelayParent),

		// Buffer size 2: one active delay + safety margin
		availableInherent: make(chan common.Hash, 2),
	}
}

type Provisioner struct {
	perRelayParent map[common.Hash]perRelayParent

	// TODO #4162
	// This doesn't have to be a channel with buffer.
	// The idea is to send a relay parent hash on this channel after INHERENT_TIMEOUT, open to design changes
	availableInherent chan common.Hash
}

func (p *Provisioner) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			err := p.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %s", err)
			}
		case <-p.availableInherent:
			// This inherentAfterDelay gets populated while handling active leaves update signal
			// TODO #4162
		}

	}
}

func (p *Provisioner) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := p.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			logger.Errorf("processing active leaves update signal: %s", err)
		}
	case provisionermessages.RequestInherentData:
		// TODO #4159
	case provisionermessages.ProvisionableData:
		// TODO #4160
	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil

}

func (*Provisioner) Name() parachaintypes.SubSystemName {
	return parachaintypes.Provisioner
}

func (p *Provisioner) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	for _, deactivated := range update.Deactivated {
		delete(p.perRelayParent, deactivated)
	}

	if update.Activated != nil {
		p.perRelayParent[update.Activated.Hash] = perRelayParent{leaf: update.Activated}

		go func() {
			time.Sleep(inherentPreProposeTimeout)
			p.availableInherent <- update.Activated.Hash
		}()
	}
	return nil
}

func (*Provisioner) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (*Provisioner) Stop() {}

type perRelayParent struct {
	leaf             *parachaintypes.ActivatedLeaf
	signedBitFields  []parachaintypes.CheckedSignedAvailabilityBitfield //nolint:unused
	isInherentReady  bool                                               //nolint:unused
	awaitingInherent []chan provisionermessages.ProvisionerInherentData //nolint:unused
}
