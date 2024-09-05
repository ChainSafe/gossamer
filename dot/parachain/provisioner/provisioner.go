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
	// TODO This doesn't have to be a channel with buffer.
	// The idea is to send a relay parent hash on this channel after INHERENT_TIMEOUT, open to design changes
	inherentAfterDelay chan common.Hash
}

func (p Provisioner) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	// TODO https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L177

	for {
		select {
		// TODO: polkadot-rust changes reputation in batches, so we do the same?
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
			// TODO https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L181
		}

	}
}

func (p Provisioner) processMessage(msg any) error {
	switch msg.(type) {
	case provisionermessages.RequestInherentData:
		// TODO https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L253
	case provisionermessages.ProvisionableData:
		// TODO https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L271
	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil

}

func (p Provisioner) Name() parachaintypes.SubSystemName {
	return parachaintypes.Provisioner
}

func (p Provisioner) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L173
	// https://github.com/paritytech/polkadot-sdk/blob/e1460b5ee5f4490b428035aa4a72c1c99a262459/polkadot/node/core/provisioner/src/lib.rs#L201
	return nil
}

func (p Provisioner) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (p Provisioner) Stop() {}

type RequestInherentData struct {
	RelayParent             common.Hash
	ProvisionerInherentData chan ProvisionerInherentData
}

type ProvisionerInherentData struct {
}

type ProvisionableData struct {
	RelayParent common.Hash
}
