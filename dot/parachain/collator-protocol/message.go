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

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol scale.VaryingDataType

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	vdt := scale.MustNewVaryingDataType(NewCollatorProtocolMessage())
	return CollationProtocol(vdt)
}

// New will enable scale to create new instance when needed
func (CollationProtocol) New() CollationProtocol {
	return NewCollationProtocol()
}

// Set will set a value using the underlying  varying data type
func (c *CollationProtocol) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollationProtocol(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *CollationProtocol) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// CollatorProtocolMessage represents Network messages used by the collator protocol subsystem
type CollatorProtocolMessage scale.VaryingDataType

// Index returns the index of varying data type
func (CollatorProtocolMessage) Index() uint {
	return 0
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewCollatorProtocolMessage() CollatorProtocolMessage {
	vdt := scale.MustNewVaryingDataType(Declare{}, AdvertiseCollation{}, CollationSeconded{})
	return CollatorProtocolMessage(vdt)
}

// New will enable scale to create new instance when needed
func (CollatorProtocolMessage) New() CollatorProtocolMessage {
	return NewCollatorProtocolMessage()
}

// Set will set a value using the underlying  varying data type
func (c *CollatorProtocolMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*c)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	*c = CollatorProtocolMessage(vdt)
	return
}

// Value returns the value from the underlying varying data type
func (c *CollatorProtocolMessage) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*c)
	return vdt.Value()
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        parachaintypes.CollatorID        `scale:"1"`
	ParaID            uint32                           `scale:"2"`
	CollatorSignature parachaintypes.CollatorSignature `scale:"3"`
}

// Index returns the index of varying data type
func (Declare) Index() uint {
	return 0
}

// AdvertiseCollation contains a relay parent hash and is used to advertise a collation to a validator.
// This will only advertise a collation if there exists one for the given relay parent and the given peer is
// set as validator for our para at the given relay parent.
// It can only be sent once the peer has declared that they are a collator with given ID
type AdvertiseCollation common.Hash

// Index returns the index of varying data type
func (AdvertiseCollation) Index() uint {
	return 1
}

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
// membership gets recognized before relay parent goes out of view.
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
func (cpvs CollatorProtocolValidatorSide) enqueueCollation(collations Collations) {
	switch collations.status {
	case Fetching, WaitingOnValidation:

	}

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

	isAdvertisementInvalid, err := peerData.InsertAdvertisement(
		relayParent,
		perRelayParent.prospectiveParachainMode,
		&prospectiveCandidate.CandidateHash,
		cpvs.implicitView,
		cpvs.activeLeaves,
	)
	if isAdvertisementInvalid {
		cpvs.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.UnexpectedMessageValue,
			Reason: peerset.UnexpectedMessageReason,
		}, sender)
		logger.Errorf(ErrInvalidAdvertisement.Error())
	}
	if err != nil {
		return fmt.Errorf("inserting advertisement: %w", err)
	}

	if perRelayParent.collations.IsSecondedLimitReached(perRelayParent.prospectiveParachainMode) {
		return ErrSecondedLimitReached
	}

	isSecondingAllowed := !perRelayParent.prospectiveParachainMode.isEnabled || cpvs.canSecond(
		collatorParaID,
		relayParent,
		prospectiveCandidate.CandidateHash,
		prospectiveCandidate.ParentHeadDataHash,
	)

	if !isSecondingAllowed {
		logger.Infof("Seconding is not allowed by backing, queueing advertisement, relay parent: %s, para id: %d, candidate hash: %s",
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

	cpvs.enqueueCollation(perRelayParent.collations)

	return nil
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

	collatorProtocolMessageV, err := collatorProtocolMessage.Value()
	if err != nil {
		return propagate, fmt.Errorf("getting collator protocol message value: %w", err)
	}

	switch collatorProtocolMessage.Index() {
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
