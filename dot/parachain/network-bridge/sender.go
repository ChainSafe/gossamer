// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package networkbridge

import (
	"context"
	"fmt"

	collatorprotocol "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
)

type NetworkBridgeSender struct {
	OverseerToSubSystem <-chan any
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

func (nbs *NetworkBridgeSender) processMessage(msg any) error { //nolint
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case SendCollationMessage:
		// TODO
		fmt.Println(msg)
	case SendValidationMessage:
		// TODO: add SendValidationMessages and SendCollationMessages to send multiple messages at the same time
		// TODO: add ConnectTOResolvedValidators, SendRequests
	case ConnectToValidators:
		// TODO
	case ReportPeer:
		// TODO
	case DisconnectPeer:
		// TODO
	}

	return nil
}

// NOTE: This is not same as corresponding rust structure
// TODO: If need be, add ability to report multiple peers in batches
type ReportPeer struct {
	peerID           peer.ID                  //nolint
	reputationChange peerset.ReputationChange //nolint
}

type DisconnectPeer struct {
	peer    peer.ID     //nolint
	peerSet peerSetType //nolint
}

type SendCollationMessage struct {
	to []peer.ID //nolint
	// TODO: make this versioned
	collationProtocolMessage collatorprotocol.CollationProtocol //nolint
}

type SendValidationMessage struct {
	to []peer.ID //nolint
	// TODO: move validation protocol to a new package to be able to be used here
	// validationProtocolMessage
}

type peerSetType int //nolint

const (
	validationProtocol peerSetType = iota //nolint
	collationProtocol                     //nolint
)

type ConnectToValidators struct {
	// IDs of the validators to connect to.
	validatorIDs []parachaintypes.AuthorityDiscoveryID //nolint
	// The underlying protocol to use for this request.
	peerSet peerSetType //nolint
	// Sends back the number of `AuthorityDiscoveryId`s which
	// authority discovery has failed to resolve.
	failed chan<- uint //nolint
}
