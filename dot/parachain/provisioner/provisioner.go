package provisioner

import (
	"context"
	"time"

	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "provisioner"))

// we expect an inherent to be ready after this time.
const INHERENT_TIMEOUT = time.Millisecond * 2000

type Provisioner struct {
	// TODO #4162
	// This doesn't have to be a channel with buffer.
	// The idea is to send a relay parent hash on this channel after INHERENT_TIMEOUT, open to design changes
	availableInherent chan common.Hash
}

func (p Provisioner) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			err := p.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case <-p.inherentAfterDelay:
			// This inherentAfterDelay gets populated while handling active leaves update signal
			// TODO #4162
		}

	}
}

func (p Provisioner) processMessage(msg any) error {
	switch msg.(type) {
	case provisionermessages.RequestInherentData:
		// TODO #4159
	case provisionermessages.ProvisionableData:
		// TODO #4160
	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil

}

func (p Provisioner) Name() parachaintypes.SubSystemName {
	return parachaintypes.Provisioner
}

func (p Provisioner) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO #4061
	return nil
}

func (p Provisioner) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (p Provisioner) Stop() {}
