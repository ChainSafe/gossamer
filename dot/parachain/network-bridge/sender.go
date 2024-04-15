package networkbridge

import (
	"context"
	"fmt"

	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type NetworkBridgeSender struct {
	net                  Network
	OverseerToSubSystem  <-chan any
	SubsystemsToOverseer chan<- any
}

func Register(overseerChan chan<- any, net Network) *NetworkBridgeSender {
	return &NetworkBridgeSender{
		net:                  net,
		SubsystemsToOverseer: overseerChan,
	}
}

func (nbs *NetworkBridgeSender) Run(ctx context.Context, OverseerToSubSystem chan any,
	SubSystemToOverseer chan any) {

	for msg := range nbs.OverseerToSubSystem {
		err := nbs.processMessage(msg)
		if err != nil {
			logger.Errorf("processing overseer message: %w", err)
		}
	}
}

func (nbs *NetworkBridgeSender) Name() parachaintypes.SubSystemName {
	return parachaintypes.NetworkBridgeSender
}

func (nbs *NetworkBridgeSender) ProcessActiveLeavesUpdateSignal() {}

func (nbs *NetworkBridgeSender) ProcessBlockFinalizedSignal() {}

func (nbs *NetworkBridgeSender) Stop() {}

func (nbs *NetworkBridgeSender) processMessage(msg any) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case networkbridgemessages.SendCollationMessage:
		// TODO
		fmt.Println(msg)
	case networkbridgemessages.SendValidationMessage:
		// TODO: add SendValidationMessages and SendCollationMessages to send multiple messages at the same time
		// TODO: add ConnectTOResolvedValidators, SendRequests
	case networkbridgemessages.ConnectToValidators:
		// TODO
	case networkbridgemessages.ReportPeer:
		nbs.net.ReportPeer(msg.ReputationChange, msg.PeerID)
	case networkbridgemessages.DisconnectPeer:
		// TODO
	}

	return nil
}
