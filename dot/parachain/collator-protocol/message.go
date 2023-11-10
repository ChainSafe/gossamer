// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

const legacyCollationProtocolV1 = "/polkadot/collation/1"

type CollationProtocolValues interface {
	CollatorProtocolMessage
}

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol struct {
	inner any
}

func setCollationProtocol[Value CollationProtocolValues](mvdt *CollationProtocol, value Value) {
	mvdt.inner = value
}

func (mvdt *CollationProtocol) SetValue(value any) (err error) {
	switch value := value.(type) {
	case CollatorProtocolMessage:
		setCollationProtocol(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationProtocol) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case CollatorProtocolMessage:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollationProtocol) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollationProtocol) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(CollatorProtocolMessage), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	return CollationProtocol{}
}

type CollatorProtocolMessageValues interface {
	Declare | AdvertiseCollation | CollationSeconded
}

// CollatorProtocolMessage represents Network messages used by the collator protocol subsystem
type CollatorProtocolMessage struct {
	inner any
}

func setCollatorProtocolMessage[Value CollatorProtocolMessageValues](mvdt *CollatorProtocolMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *CollatorProtocolMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Declare:
		setCollatorProtocolMessage(mvdt, value)
		return

	case AdvertiseCollation:
		setCollatorProtocolMessage(mvdt, value)
		return

	case CollationSeconded:
		setCollatorProtocolMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollatorProtocolMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Declare:
		return 0, mvdt.inner, nil

	case AdvertiseCollation:
		return 1, mvdt.inner, nil

	case CollationSeconded:
		return 4, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollatorProtocolMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollatorProtocolMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Declare), nil

	case 1:
		return *new(AdvertiseCollation), nil

	case 4:
		return *new(CollationSeconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewCollatorProtocolMessage() CollatorProtocolMessage {
	return CollatorProtocolMessage{}
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        parachaintypes.CollatorID        `scale:"1"`
	ParaID            uint32                           `scale:"2"`
	CollatorSignature parachaintypes.CollatorSignature `scale:"3"`
}

// AdvertiseCollation contains a relay parent hash and is used to advertise a collation to a validator.
// This will only advertise a collation if there exists one for the given relay parent and the given peer is
// set as validator for our para at the given relay parent.
// It can only be sent once the peer has declared that they are a collator with given ID
type AdvertiseCollation common.Hash

// CollationSeconded represents that a collation sent to a validator was seconded.
type CollationSeconded struct {
	RelayParent common.Hash                                 `scale:"1"`
	Statement   parachaintypes.UncheckedSignedFullStatement `scale:"2"`
}

// Index returns the index of varying data type
func (CollationSeconded) Index() uint {
	return 4
}

const MaxCollationMessageSize uint64 = 100 * 1024

// Type returns CollationMsgType
func (CollationProtocol) Type() network.MessageType {
	return network.CollationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (cp CollationProtocol) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (cp CollationProtocol) Encode() ([]byte, error) {
	enc, err := scale.Marshal(cp)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	collationMessage := CollationProtocol{}

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

// BlockedAdvertisement is vstaging advertisement that was rejected by the backing
// subsystem. Validator may fetch it later if its fragment
// membership gets recognised before relay parent goes out of view.
type BlockedAdvertisement struct {
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
) bool {
	canSecondRequest := backing.CanSecond{
		CandidateParaID:      candidateParaID,
		CandidateRelayParent: candidateRelayParent,
		CandidateHash:        candidateHash,
		ParentHeadDataHash:   parentHeadDataHash,
	}

	responseChan := make(chan bool)

	cpvs.SubSystemToOverseer <- struct {
		responseChan     chan bool
		canSecondRequest backing.CanSecond
	}{
		responseChan:     responseChan,
		canSecondRequest: canSecondRequest,
	}

	// TODO: Add timeout
	return <-responseChan
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
		if perRelayParent.prospectiveParachainMode.isEnabled {
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
	// TODO:
	// - tracks advertisements received and the source (peer id) of the advertisement
	// - accept one advertisement per collator per source per relay-parent

	perRelayParent, ok := cpvs.perRelayParent[relayParent]
	if !ok {
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
		return ErrRelayParentUnknown
	}

	peerData, ok := cpvs.peerData[sender]
	if !ok {
		return ErrUnknownPeer
	}

	if peerData.state.PeerState != Collating {
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
		return ErrUndeclaredPara
	}

	collatorParaID := peerData.state.CollatingPeerState.ParaID

	if perRelayParent.assignment == nil || *perRelayParent.assignment != collatorParaID {
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.WrongParaValue,
			Reason: peerset.WrongParaReason,
		}, sender)
		return ErrInvalidAssignment
	}

	// Note: Prospective Parachain mode would be set or edited when the view gets updated.
	if perRelayParent.prospectiveParachainMode.isEnabled && prospectiveCandidate == nil {
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
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
		logger.Errorf(ErrInvalidAdvertisement.Error())
	} else if err != nil {
		return fmt.Errorf("inserting advertisement: %w", err)
	}

	if perRelayParent.collations.IsSecondedLimitReached(perRelayParent.prospectiveParachainMode) {
		return ErrSecondedLimitReached
	}

	/*NOTE:---------------------------------------Matters only in V2----------------------------------------------*/
	isSecondingAllowed := !perRelayParent.prospectiveParachainMode.isEnabled || cpvs.canSecond(
		collatorParaID,
		relayParent,
		prospectiveCandidate.CandidateHash,
		prospectiveCandidate.ParentHeadDataHash,
	)

	if !isSecondingAllowed {
		logger.Infof("Seconding is not allowed by backing, queueing advertisement,"+
			" relay parent: %s, para id: %d, candidate hash: %s",
			relayParent, collatorParaID, prospectiveCandidate.CandidateHash)

		blockedAdvertisements := append(cpvs.BlockedAdvertisements, BlockedAdvertisement{
			peerID:               sender,
			collatorID:           peerData.state.CollatingPeerState.CollatorID,
			candidateRelayParent: relayParent,
			candidateHash:        prospectiveCandidate.CandidateHash,
		})

		cpvs.BlockedAdvertisements = blockedAdvertisements
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

func (cpvs CollatorProtocolValidatorSide) handleCollationMessage(
	sender peer.ID, msg network.NotificationsMessage) (bool, error) {

	// we don't propagate collation messages, so it will always be false
	propagate := false

	if msg.Type() != network.CollationMsgType {
		return propagate, fmt.Errorf("%w, expected: %d, found:%d", ErrUnexpectedMessageOnCollationProtocol,
			network.CollationMsgType, msg.Type())
	}

	collatorProtocol, ok := msg.(*CollationProtocol)
	if !ok {
		return propagate, fmt.Errorf(
			"failed to cast into collator protocol message, expected: *CollationProtocol, got: %T",
			msg)
	}

	collatorProtocolV, err := collatorProtocol.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolV.(CollatorProtocolMessage)
	if !ok {
		return propagate, errors.New("expected value to be collator protocol message")
	}

	index, collatorProtocolMessageV, err := collatorProtocolMessage.IndexValue()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol message value: %w", err)
	}

	switch index {
	// TODO: Create an issue to cover v2 types. #3534
	case 0: // Declare
		declareMessage, ok := collatorProtocolMessageV.(Declare)
		if !ok {
			return propagate, errors.New("expected message to be declare")
		}

		// check if we already have the collator id declared in this message. If so, punish the
		// peer who sent us this message by reducing its reputation
		_, ok = cpvs.getPeerIDFromCollatorID(declareMessage.CollatorId)
		if ok {
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			}, sender)
			return propagate, nil
		}

		// NOTE: peerData for sender will be filled when it gets connected to us
		peerData, ok := cpvs.peerData[sender]
		if !ok {
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			}, sender)
			return propagate, fmt.Errorf("%w: %s", ErrUnknownPeer, sender)
		}

		if peerData.state.PeerState == Collating {
			logger.Error("peer is already in the collating state")
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			}, sender)
			return propagate, nil
		}

		// check signature declareMessage.CollatorSignature
		err = sr25519.VerifySignature(declareMessage.CollatorId[:], declareMessage.CollatorSignature[:],
			getDeclareSignaturePayload(sender))
		if errors.Is(err, crypto.ErrSignatureVerificationFailed) {
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.InvalidSignatureValue,
				Reason: peerset.InvalidSignatureReason,
			}, sender)
			return propagate, fmt.Errorf("invalid signature: %w", err)
		}
		if err != nil {
			return propagate, fmt.Errorf("verifying signature: %w", err)
		}

		// NOTE: assignments are setting when we handle view changes
		_, ok = cpvs.currentAssignments[parachaintypes.ParaID(declareMessage.ParaID)]
		if ok {
			logger.Errorf("declared as collator for current para: %d", declareMessage.ParaID)

			peerData.SetCollating(declareMessage.CollatorId, parachaintypes.ParaID(declareMessage.ParaID))
			cpvs.peerData[sender] = peerData
		} else {
			logger.Errorf("declared as collator for unneeded para: %d", declareMessage.ParaID)
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnneededCollatorValue,
				Reason: peerset.UnneededCollatorReason,
			}, sender)

			// TODO: Disconnect peer. #3530
			// Do a thorough review of substrate/client/network/src/
			// check how are they managing peerset of different protocol.
			// Currently we have a Handler in dot/peerset, but it does not get used anywhere.
		}
	case 1: // AdvertiseCollation
		advertiseCollationMessage, ok := collatorProtocolMessageV.(AdvertiseCollation)
		if !ok {
			return propagate, errors.New("expected message to be advertise collation")
		}

		err := cpvs.handleAdvertisement(common.Hash(advertiseCollationMessage), sender, nil)
		if err != nil {
			return propagate, fmt.Errorf("handling v1 advertisement: %w", err)
		}
		// TODO:
		// - tracks advertisements received and the source (peer id) of the advertisement
		// - accept one advertisement per collator per source per relay-parent

	case 2: // CollationSeconded
		logger.Errorf("unexpected collation seconded message from peer %s, decreasing its reputation", sender)
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
	}

	return propagate, nil
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
