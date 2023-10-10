// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/exp/slices"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "collator-protocol"))

const (
	activityPoll            = 10 * time.Millisecond
	maxUnsharedDownloadTime = 100 * time.Millisecond
)

var (
	ErrUnexpectedMessageOnCollationProtocol = errors.New("unexpected message on collation protocol")
	ErrUnknownPeer                          = errors.New("unknown peer")
	ErrUnknownOverseerMessage               = errors.New("unknown overseer message type")
	ErrNotExpectedOnValidatorSide           = errors.New("message is not expected on the validator side of the protocol")
	ErrCollationNotInView                   = errors.New("collation is not in our view")
	ErrPeerIDNotFoundForCollator            = errors.New("peer id not found for collator")
)

func (cpvs CollatorProtocolValidatorSide) Run(
	ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error {
	inactivityTicker := time.NewTicker(activityPoll)

	for {
		select {
		// TODO: polkadot-rust changes reputation in batches, so we do the same?
		case msg, ok := <-cpvs.OverseerToSubSystem:
			if !ok {
				return nil
			}

			err := cpvs.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %w", err)
			}
		case <-inactivityTicker.C:
			// TODO: disconnect inactive peers
			// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L1301

		case unfetchedCollation := <-cpvs.unfetchedCollation:
			// TODO: If we can't get the collation from given collator within MAX_UNSHARED_DOWNLOAD_TIME,
			// we will start another one from the next collator.

			// check if this peer id has advertised this relay parent
			peerData := cpvs.peerData[unfetchedCollation.PendingCollation.PeerID]
			if peerData.HasAdvertisedRelayParent(unfetchedCollation.PendingCollation.RelayParent) {
				// if so request collation from this peer id
				collation, err := cpvs.requestCollation(unfetchedCollation.PendingCollation.RelayParent,
					unfetchedCollation.PendingCollation.ParaID, unfetchedCollation.PendingCollation.PeerID)
				if err != nil {
					logger.Errorf("fetching collation: %w", err)
				}
				cpvs.fetchedCollations = append(cpvs.fetchedCollations, *collation)
			}

		}
	}
}

// requestCollation requests a collation from the network.
// This function will
// - check for duplicate requests
// - check if the requested collation is in our view
func (cpvs CollatorProtocolValidatorSide) requestCollation(relayParent common.Hash,
	paraID parachaintypes.ParaID, peerID peer.ID) (*parachaintypes.Collation, error) {
	if !slices.Contains[[]common.Hash](cpvs.ourView.heads, relayParent) {
		return nil, ErrCollationNotInView
	}

	// make collation fetching request
	collationFetchingRequest := CollationFetchingRequest{
		RelayParent: relayParent,
		ParaID:      paraID,
	}

	collationFetchingResponse := NewCollationFetchingResponse()
	err := cpvs.collationFetchingReqResProtocol.Do(peerID, collationFetchingRequest, &collationFetchingResponse)
	if err != nil {
		return nil, fmt.Errorf("collation fetching request failed: %w", err)
	}

	v, err := collationFetchingResponse.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value of collation fetching response: %w", err)
	}
	collationVDT, ok := v.(CollationVDT)
	if !ok {
		return nil, fmt.Errorf("collation fetching response value expected: CollationVDT, got: %T", v)
	}
	collation := parachaintypes.Collation(collationVDT)

	return &collation, nil
}

type UnfetchedCollation struct {
	CollatorID       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type PendingCollation struct {
	RelayParent    common.Hash
	ParaID         parachaintypes.ParaID
	PeerID         peer.ID
	CommitmentHash *common.Hash
}

type PeerData struct {
	view  View
	state PeerStateInfo
}

func (peerData *PeerData) HasAdvertisedRelayParent(relayParent common.Hash) bool {
	if peerData.state.PeerState == Connected {
		return false
	}

	return slices.Contains(peerData.view.heads, relayParent)
}

func (peerData *PeerData) SetCollating(collatorID parachaintypes.CollatorID, paraID parachaintypes.ParaID) {
	peerData.state = PeerStateInfo{
		PeerState: Collating,
		CollatingPeerState: CollatingPeerState{
			CollatorID: collatorID,
			ParaID:     paraID,
		},
	}
}

func (peerData *PeerData) InsertAdvertisement() error {
	// TODO: part of https://github.com/ChainSafe/gossamer/issues/3514
	return nil
}

type PeerStateInfo struct {
	PeerState PeerState
	// instant at which peer got connected
	Instant            time.Time
	CollatingPeerState CollatingPeerState
}

type CollatingPeerState struct {
	CollatorID     parachaintypes.CollatorID
	ParaID         parachaintypes.ParaID
	advertisements []common.Hash //nolint
	lastActive     time.Time     //nolint
}

type PeerState uint

const (
	Connected PeerState = iota
	Collating
)

type View struct {
	// a bounded amount of chain heads
	heads []common.Hash
	// the highest known finalized number
	finalizedNumber uint32 //nolint
}

// Network is the interface required by parachain service for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg network.NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID network.MessageType,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
	GetRequestResponseProtocol(subprotocol string, requestTimeout time.Duration,
		maxResponseSize uint64) *network.RequestResponseProtocol
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

type CollationEvent struct {
	CollatorId       parachaintypes.CollatorID
	PendingCollation PendingCollation
}

type CollatorProtocolValidatorSide struct {
	net Network

	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any

	unfetchedCollation chan UnfetchedCollation

	collationFetchingReqResProtocol *network.RequestResponseProtocol

	fetchedCollations []parachaintypes.Collation
	// track all active collators and their data
	peerData map[peer.ID]PeerData

	ourView View

	// Keep track of all pending candidate collations
	pendingCandidates map[common.Hash]CollationEvent

	// Parachains we're currently assigned to. With async backing enabled
	// this includes assignments from the implicit view.
	currentAssignments map[parachaintypes.ParaID]uint
}

func (cpvs CollatorProtocolValidatorSide) getPeerIDFromCollatorID(collatorID parachaintypes.CollatorID,
) (peer.ID, bool) {
	for peerID, peerData := range cpvs.peerData {
		if peerData.state.PeerState == Collating && peerData.state.CollatingPeerState.CollatorID == collatorID {
			return peerID, true
		}
	}

	return "", false
}

type CollateOn parachaintypes.ParaID

type DistributeCollation struct {
	CandidateReceipt parachaintypes.CandidateReceipt
	PoV              parachaintypes.PoV
}

type ReportCollator parachaintypes.CollatorID

type NetworkBridgeUpdate struct {
	// TODO: not quite sure if we would need this or something similar to this
}

type SecondedOverseerMsg struct {
	Parent common.Hash
	Stmt   parachaintypes.StatementVDT
}

type Backed struct {
	ParaID parachaintypes.ParaID
	// Hash of the para head generated by candidate
	ParaHead common.Hash
}

type InvalidOverseeMsg struct {
	Parent           common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
}

func (cpvs CollatorProtocolValidatorSide) processMessage(msg interface{}) error {
	// run this function as a goroutine, ideally

	switch msg := msg.(type) {
	case CollateOn:
		return fmt.Errorf("CollateOn %w", ErrNotExpectedOnValidatorSide)
	case DistributeCollation:
		return fmt.Errorf("DistributeCollation %w", ErrNotExpectedOnValidatorSide)
	case ReportCollator:
		peerID, ok := cpvs.getPeerIDFromCollatorID(parachaintypes.CollatorID(msg))
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)
	case NetworkBridgeUpdate:
		// TODO: handle network message https://github.com/ChainSafe/gossamer/issues/3515
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1457 //nolint
	case SecondedOverseerMsg:
		// TODO: handle seconded message https://github.com/ChainSafe/gossamer/issues/3516
		// https://github.com/paritytech/polkadot-sdk/blob/db3fd687262c68b115ab6724dfaa6a71d4a48a59/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L1466 //nolint

	case Backed:
		// TODO: handle backed message https://github.com/ChainSafe/gossamer/issues/3517
	case InvalidOverseeMsg:
		invalidOverseerMsg := msg

		collationEvent, ok := cpvs.pendingCandidates[invalidOverseerMsg.Parent]
		if !ok {
			return nil
		}

		if *collationEvent.PendingCollation.CommitmentHash == (invalidOverseerMsg.CandidateReceipt.CommitmentsHash) {
			delete(cpvs.pendingCandidates, invalidOverseerMsg.Parent)
		} else {
			logger.Error("reported invalid candidate for unknown `pending_candidate`")
			return nil
		}

		peerID, ok := cpvs.getPeerIDFromCollatorID(collationEvent.CollatorId)
		if !ok {
			return ErrPeerIDNotFoundForCollator
		}
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.ReportBadCollatorValue,
			Reason: peerset.ReportBadCollatorReason,
		}, peerID)
	default:
		return ErrUnknownOverseerMessage
	}

	return nil
}
