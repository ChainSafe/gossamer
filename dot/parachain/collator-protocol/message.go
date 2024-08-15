// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	collationMessage := collatorprotocolmessages.CollationProtocol{}

	err := scale.Unmarshal(in, &collationMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &collationMessage, nil
}

type ProspectiveCandidate struct {
	CandidateHash      parachaintypes.CandidateHash
	ParentHeadDataHash common.Hash
}

type CollationStatus int

const (
	// We are waiting for a collation to be advertised to us.
	Waiting CollationStatus = iota
	// We are currently fetching a collation.
	Fetching
	// We are waiting that a collation is being validated.
	WaitingOnValidation
	// We have seconded a collation.
	Seconded
)

// blockedAdvertisement is vstaging advertisement that was rejected by the backing
// subsystem. Validator may fetch it later if its fragment
// membership gets recognised before relay parent goes out of view.
type blockedAdvertisement struct {
	// peer that advertised the collation
	peerID               peer.ID
	collatorID           parachaintypes.CollatorID
	candidateRelayParent common.Hash
	candidateHash        parachaintypes.CandidateHash
}

func (cpvs CollatorProtocolValidatorSide) canSecond(
	candidateParaID parachaintypes.ParaID,
	candidateRelayParent common.Hash,
	candidateHash parachaintypes.CandidateHash,
	parentHeadDataHash common.Hash,
) (bool, error) {

	canSecondRequest := backing.CanSecondMessage{
		CandidateParaID:      candidateParaID,
		CandidateRelayParent: candidateRelayParent,
		CandidateHash:        candidateHash,
		ParentHeadDataHash:   parentHeadDataHash,
		ResponseCh:           make(chan bool),
	}

	cpvs.SubSystemToOverseer <- canSecondRequest
	select {
	case canSecondResponse := <-canSecondRequest.ResponseCh:
		return canSecondResponse, nil
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return false, parachaintypes.ErrSubsystemRequestTimeout
	}
}

// Enqueue collation for fetching. The advertisement is expected to be
// validated.
func (cpvs CollatorProtocolValidatorSide) enqueueCollation(
	collations Collations,
	relayParent common.Hash,
	paraID parachaintypes.ParaID,
	peerID peer.ID,
	collatorID parachaintypes.CollatorID,
	prospectiveCandidate *ProspectiveCandidate) error {

	// TODO: return errors
	pendingCollation := PendingCollation{
		RelayParent:          relayParent,
		ParaID:               paraID,
		PeerID:               peerID,
		ProspectiveCandidate: prospectiveCandidate,
	}

	switch collations.status {
	// TODO: In rust code, a lot of thing that are being done in handle_advertisement
	// are being repeated here.
	// Currently enqueueCollation is being called from handle_advertisement only, so we might not need to
	// repeat that here.
	// If enqueueCollation gets used somewhere else, we would need to repeat those things here.

	case Fetching, WaitingOnValidation:
		logger.Debug("added collation to unfetched list")
		collations.waitingQueue = append(collations.waitingQueue, UnfetchedCollation{
			CollatorID:       collatorID,
			PendingCollation: pendingCollation,
		})
	case Waiting:
		// limit is not reached, it's allowed to second another collation
		return cpvs.fetchCollation(pendingCollation)
	case Seconded:
		perRelayParent, ok := cpvs.perRelayParent[relayParent]
		if !ok {
			logger.Error("candidate relay parent went out of view for valid advertisement")
			return ErrRelayParentUnknown
		}
		if perRelayParent.prospectiveParachainMode.IsEnabled {
			return cpvs.fetchCollation(pendingCollation)
		} else {
			logger.Debug("a collation has already been seconded")
		}
	}

	return nil
}

func (cpvs *CollatorProtocolValidatorSide) fetchCollation(pendingCollation PendingCollation) error {
	var candidateHash *parachaintypes.CandidateHash
	if pendingCollation.ProspectiveCandidate != nil {
		candidateHash = &pendingCollation.ProspectiveCandidate.CandidateHash
	}
	peerData, ok := cpvs.peerData[pendingCollation.PeerID]
	if !ok {
		return ErrUnknownPeer
	}

	if !peerData.HasAdvertised(pendingCollation.RelayParent, candidateHash) {
		return ErrNotAdvertised
	}

	// TODO: Add it to collation_fetch_timeouts if we can't process this in timeout time.
	// state
	// .collation_fetch_timeouts
	// .push(timeout(id.clone(), candidate_hash, relay_parent).boxed());
	collation, err := cpvs.requestCollation(pendingCollation.RelayParent, pendingCollation.ParaID,
		pendingCollation.PeerID)
	if err != nil {
		return fmt.Errorf("requesting collation: %w", err)
	}

	cpvs.fetchedCollations = append(cpvs.fetchedCollations, *collation)

	return nil
}

func (cpvs *CollatorProtocolValidatorSide) handleAdvertisement(relayParent common.Hash, sender peer.ID,
	prospectiveCandidate *ProspectiveCandidate) error {
	perRelayParent, ok := cpvs.perRelayParent[relayParent]
	if !ok {
		cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
			PeerID: sender,
			ReputationChange: peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			},
		}

		return ErrRelayParentUnknown
	}

	peerData, ok := cpvs.peerData[sender]
	if !ok {
		return ErrUnknownPeer
	}

	if peerData.state.PeerState != Collating {
		cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
			PeerID: sender,
			ReputationChange: peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			},
		}

		return ErrUndeclaredPara
	}

	collatorParaID := peerData.state.CollatingPeerState.ParaID

	if perRelayParent.assignment == nil || *perRelayParent.assignment != collatorParaID {
		cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
			PeerID: sender,
			ReputationChange: peerset.ReputationChange{
				Value:  peerset.WrongParaValue,
				Reason: peerset.WrongParaReason,
			},
		}

		return ErrInvalidAssignment
	}

	// Note: Prospective Parachain mode would be set or edited when the view gets updated.
	if perRelayParent.prospectiveParachainMode.IsEnabled && prospectiveCandidate == nil {
		// Expected v2 advertisement.
		return ErrProtocolMismatch
	}

	var prospectiveCandidateHash *parachaintypes.CandidateHash
	if prospectiveCandidate != nil {
		prospectiveCandidateHash = &prospectiveCandidate.CandidateHash
	}

	isAdvertisementValid, err := peerData.InsertAdvertisement(
		relayParent,
		perRelayParent.prospectiveParachainMode,
		prospectiveCandidateHash,
		cpvs.implicitView,
		cpvs.activeLeaves,
	)
	if !isAdvertisementValid {
		cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
			PeerID: sender,
			ReputationChange: peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			},
		}

		logger.Errorf(ErrInvalidAdvertisement.Error())
	} else if err != nil {
		return fmt.Errorf("inserting advertisement: %w", err)
	}

	if perRelayParent.collations.IsSecondedLimitReached(perRelayParent.prospectiveParachainMode) {
		return ErrSecondedLimitReached
	}

	/*NOTE:---------------------------------------Matters only in V2----------------------------------------------*/
	var isSecondingAllowed bool
	if !perRelayParent.prospectiveParachainMode.IsEnabled {
		isSecondingAllowed = true
	} else {
		isSecondingAllowed, err = cpvs.canSecond(
			collatorParaID,
			relayParent,
			prospectiveCandidate.CandidateHash,
			prospectiveCandidate.ParentHeadDataHash,
		)
		if err != nil {
			return fmt.Errorf("checking if seconding is allowed: %w", err)
		}
	}

	if !isSecondingAllowed {
		logger.Infof("Seconding is not allowed by backing, queueing advertisement,"+
			" relay parent: %s, para id: %d, candidate hash: %s",
			relayParent, collatorParaID, prospectiveCandidate.CandidateHash)

		backed := collatorprotocolmessages.Backed{
			ParaID:   collatorParaID,
			ParaHead: prospectiveCandidate.ParentHeadDataHash,
		}

		blockedAd := blockedAdvertisement{
			peerID:               sender,
			collatorID:           peerData.state.CollatingPeerState.CollatorID,
			candidateRelayParent: relayParent,
			candidateHash:        prospectiveCandidate.CandidateHash,
		}
		cpvs.BlockedAdvertisements[backed.String()] = []blockedAdvertisement{blockedAd}
		return nil
	}
	/*--------------------------------------------END----------------------------------------------------------*/

	return cpvs.enqueueCollation(perRelayParent.collations,
		relayParent,
		collatorParaID,
		sender,
		peerData.state.CollatingPeerState.CollatorID,
		prospectiveCandidate)

}

// getDeclareSignaturePayload gives the payload that should be signed and included in a Declare message.
// The payload is a the local peer id of the node, which serves to prove that it controls the
// collator key it is declaring and intends to collate under.
func getDeclareSignaturePayload(peerID peer.ID) []byte {
	payload := []byte("COLL")
	payload = append(payload, peerID...)

	return payload
}

func (cpvs CollatorProtocolValidatorSide) processCollatorProtocolMessage(sender peer.ID,
	msg collatorprotocolmessages.CollationProtocol) error {

	collatorProtocolV, err := msg.Value()
	if err != nil {
		return fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolV.(collatorprotocolmessages.CollatorProtocolMessage)
	if !ok {
		return errors.New("expected value to be collator protocol message")
	}

	index, collatorProtocolMessageV, err := collatorProtocolMessage.IndexValue()
	if err != nil {
		return fmt.Errorf("getting collator protocol message value: %w", err)
	}

	switch index {
	// TODO: Create an issue to cover v2 types. #3534
	case 0: // Declare
		declareMessage, ok := collatorProtocolMessageV.(collatorprotocolmessages.Declare)
		if !ok {
			return errors.New("expected message to be declare")
		}

		// check if we already have the collator id declared in this message. If so, punish the
		// peer who sent us this message by reducing its reputation
		_, ok = cpvs.getPeerIDFromCollatorID(declareMessage.CollatorId)
		if ok {
			cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
				PeerID: sender,
				ReputationChange: peerset.ReputationChange{
					Value:  peerset.UnexpectedMessageValue,
					Reason: peerset.UnexpectedMessageReason,
				},
			}

			return nil
		}

		// NOTE: peerData for sender will be filled when it gets connected to us
		peerData, ok := cpvs.peerData[sender]
		if !ok {
			cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
				PeerID: sender,
				ReputationChange: peerset.ReputationChange{
					Value:  peerset.UnexpectedMessageValue,
					Reason: peerset.UnexpectedMessageReason,
				},
			}

			return fmt.Errorf("%w: %s", ErrUnknownPeer, sender)
		}

		if peerData.state.PeerState == Collating {
			logger.Error("peer is already in the collating state")
			cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
				PeerID: sender,
				ReputationChange: peerset.ReputationChange{
					Value:  peerset.UnexpectedMessageValue,
					Reason: peerset.UnexpectedMessageReason,
				},
			}

			return nil
		}

		// check signature declareMessage.CollatorSignature
		err = sr25519.VerifySignature(declareMessage.CollatorId[:], declareMessage.CollatorSignature[:],
			getDeclareSignaturePayload(sender))
		if errors.Is(err, crypto.ErrSignatureVerificationFailed) {
			cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
				PeerID: sender,
				ReputationChange: peerset.ReputationChange{
					Value:  peerset.InvalidSignatureValue,
					Reason: peerset.InvalidSignatureReason,
				},
			}

			return fmt.Errorf("invalid signature: %w", err)
		}
		if err != nil {
			return fmt.Errorf("verifying signature: %w", err)
		}

		// NOTE: assignments are setting when we handle view changes
		_, ok = cpvs.currentAssignments[parachaintypes.ParaID(declareMessage.ParaID)]
		if ok {
			logger.Errorf("declared as collator for current para: %d", declareMessage.ParaID)

			peerData.SetCollating(declareMessage.CollatorId, parachaintypes.ParaID(declareMessage.ParaID))
			cpvs.peerData[sender] = peerData
		} else {
			logger.Errorf("declared as collator for unneeded para: %d", declareMessage.ParaID)
			cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
				PeerID: sender,
				ReputationChange: peerset.ReputationChange{
					Value:  peerset.UnneededCollatorValue,
					Reason: peerset.UnneededCollatorReason,
				},
			}

			cpvs.SubSystemToOverseer <- networkbridgemessages.DisconnectPeer{
				Peer:    sender,
				PeerSet: networkbridgemessages.CollationProtocol,
			}

			// Do a thorough review of substrate/client/network/src/
			// check how are they managing peerset of different protocol.
			// Currently we have a Handler in dot/peerset, but it does not get used anywhere.
		}
	case 1: // AdvertiseCollation
		advertiseCollationMessage, ok := collatorProtocolMessageV.(collatorprotocolmessages.AdvertiseCollation)
		if !ok {
			return errors.New("expected message to be advertise collation")
		}

		err := cpvs.handleAdvertisement(common.Hash(advertiseCollationMessage), sender, nil)
		if err != nil {
			return fmt.Errorf("handling v1 advertisement: %w", err)
		}
		// TODO:
		// - tracks advertisements received and the source (peer id) of the advertisement
		// - accept one advertisement per collator per source per relay-parent

	case 2: // CollationSeconded
		logger.Errorf("unexpected collation seconded message from peer %s, decreasing its reputation", sender)
		cpvs.SubSystemToOverseer <- networkbridgemessages.ReportPeer{
			PeerID: sender,
			ReputationChange: peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			},
		}
	}

	return nil
}

func getCollatorHandshake() (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func decodeCollatorHandshake(_ []byte) (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func validateCollatorHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

type collatorHandshake struct{}

// String formats a collatorHandshake as a string
func (*collatorHandshake) String() string {
	return "collatorHandshake"
}

// Encode encodes a collatorHandshake message using SCALE
func (*collatorHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a collatorHandshake
func (*collatorHandshake) Decode(_ []byte) error {
	return nil
}

// IsValid returns true
func (*collatorHandshake) IsValid() bool {
	return true
}
