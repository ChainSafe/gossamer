// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	ErrProtocolMismatch     = errors.New("An advertisement format doesn't match the relay parent")
	ErrSecondedLimitReached = errors.New("Para reached a limit of seconded candidates for this relay parent.")
)

const LEGACY_COLLATION_PROTOCOL_V1 = "/polkadot/collation/1"

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

// getDeclareSignaturePayload gives the payload that should be signed and included in a Declare message.
// The payload is a the local peed id of the node, which serves to prove that it controls the
// collator key it is declaring and intends to collate under.
func getDeclareSignaturePayload(peerID peer.ID) []byte {
	payload := []byte("COLL")
	payload = append(payload, peerID...)

	return payload
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

// type CollationProtocolV1 CollationProtocol

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
	fmt.Println("decoding collation message", in)

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

func (cpvs CollatorProtocolValidatorSide) handleAdvertisement(relayParent common.Hash, peerID peer.ID, prospectiveCandidate *ProspectiveCandidate) error {
	// TODO:
	// - tracks advertisements received and the source (peer id) of the advertisement

	// - accept one advertisement per collator per source per relay-parent
	perRelayParent, ok := cpvs.perRelayParent[relayParent]
	if !ok {
		return errors.New("relay parent is unknown")
	}

	peerData, ok := cpvs.peerData[peerID]
	if !ok {
		return errors.New("peer is unknown")
	}

	if peerData.state.PeerState != Collating {
		return errors.New("peer has not declared its para id")
	}

	collatorParaID := peerData.state.CollatingPeerState.ParaID

	if perRelayParent.assignment == nil || *perRelayParent.assignment != collatorParaID {
		return errors.New("we're assigned to a different para at the given relay parent.")
	}

	if perRelayParent.prospectiveParachainMode.isEnabled && prospectiveCandidate == nil {
		return ErrProtocolMismatch
	}

	err := peerData.InsertAdvertisement()
	if err != nil {
		return fmt.Errorf("inserting advertisement: %w", err)
	}

	if perRelayParent.collations.IsSecondedLimitReached(perRelayParent.prospectiveParachainMode) {
		return ErrSecondedLimitReached
	}

	isSecondingAllowed := !perRelayParent.prospectiveParachainMode.isEnabled || canSecond()

	if !isSecondingAllowed {
		logger.Infof("Seconding is not allowed by backing, queueing advertisement, relay parent: %s, para id: %d, candidate hash: %s",
			relayParent, collatorParaID, prospectiveCandidate.CandidateHash)

		cpvs.BlockedAdvertisements = append(cpvs.BlockedAdvertisements, BlockedAdvertisement{
			peerID:               peerID,
			collatorID:           peerData.state.CollatingPeerState.CollatorID,
			candidateRelayParent: relayParent,
			candidateHash:        prospectiveCandidate.CandidateHash,
		})
		return nil
	}

	cpvs.enqueueCollation(perRelayParent.collations)

	return nil
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

func (cpvs CollatorProtocolValidatorSide) enqueueCollation(collations Collations) {
	switch collations.status {
	case Fetching, WaitingOnValidation:

	}

}

func canSecond() bool {
	// TODO
	// https://github.com/paritytech/polkadot-sdk/blob/6079b6dd3aaba56ef257111fda74a57a800f16d0/polkadot/node/network/collator-protocol/src/validator_side/mod.rs#L955
	return false
}

func (cpvs CollatorProtocolValidatorSide) handleCollationMessage(sender peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a collation message", msg)

	if msg.Type() != network.CollationMsgType {
		// TODO: Adding a string method for message type might make this message more informative
		return false, fmt.Errorf("unexpected message type, expected: %d, found:%d",
			network.CollationMsgType, msg.Type())
	}

	collatorProtocol, ok := msg.(*CollationProtocol)
	if !ok {
		return false, errors.New("failed to cast into collator protocol message")
	}

	collatorProtocolV, err := collatorProtocol.Value()
	if err != nil {
		return false, fmt.Errorf("getting collator protocol value: %w", err)
	}
	collatorProtocolMessage, ok := collatorProtocolV.(CollatorProtocolMessage)
	if !ok {
		return false, errors.New("expected value to be collator protocol message")
	}

	collatorProtocolMessageV, err := collatorProtocolMessage.Value()
	if err != nil {
		return false, fmt.Errorf("getting collator protocol message value: %w", err)
	}

	// https://github.com/paritytech/polkadot/blob/8f05479e4bd61341af69f0721e617f01cbad8bb2/node/network/collator-protocol/src/validator_side/mod.rs#L814
	switch collatorProtocolMessage.Index() {
	// TODO: Make sure that V1 and VStaging both types are covered
	// All the types covered currently are V1.
	case 0: // Declare
		declareMessage, ok := collatorProtocolMessageV.(Declare)
		if !ok {
			return false, errors.New("expected message to be declare")
		}

		// check if we already have the collator id declared in this message. If so, punish the
		// peer who sent us this message by reducing its reputation
		peerID, ok := cpvs.getPeerIDFromCollatorID(declareMessage.CollatorId)
		if ok {
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			}, peerID)
			return true, nil
		}

		peerData := cpvs.peerData[sender]
		if peerData.state.PeerState == Collating {
			logger.Error("peer is already in the collating state")
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnexpectedMessageValue,
				Reason: peerset.UnexpectedMessageReason,
			}, sender)
			return true, nil
		}

		// check signature declareMessage.CollatorSignature
		err = sr25519.VerifySignature(declareMessage.CollatorId[:], declareMessage.CollatorSignature[:],
			getDeclareSignaturePayload(peerID))
		if errors.Is(err, crypto.ErrSignatureVerificationFailed) {
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.InvalidSignatureValue,
				Reason: peerset.InvalidSignatureReason,
			}, sender)
			return true, fmt.Errorf("invalid signature: %w", err)
		}
		if err != nil {
			return false, fmt.Errorf("verifying signature: %w", err)
		}

		_, ok = cpvs.currentAssignments[parachaintypes.ParaID(declareMessage.ParaID)]
		if ok {
			logger.Errorf("declared as collator for current para: %d", declareMessage.ParaID)

			// TODO: Add this
			// peer_data.set_collating(collator_id, para_id);
		} else {
			logger.Errorf("declared as collator for unneeded para: %d", declareMessage.ParaID)
			cpvs.net.ReportPeer(peerset.ReputationChange{
				Value:  peerset.UnneededCollatorValue,
				Reason: peerset.UnneededCollatorReason,
			}, sender)
			// TODO: Disconnect peer
		}
	case 1: // AdvertiseCollation
		// advertiseCollationMessage, ok := collatorProtocolMessageV.(AdvertiseCollation)
		// if !ok {
		// 	return false, errors.New("expected message to be advertise collation")
		// }

		// err := cpvs.handleAdvertisement(common.Hash(advertiseCollationMessage))
		// if err != nil {
		// 	return false, fmt.Errorf("handling advertisement: %w", err)
		// }
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

	return false, nil
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
