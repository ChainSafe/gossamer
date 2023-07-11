// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inherents

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// signature could be one of Ed25519 signature, Sr25519 signature or ECDSA/SECP256k1 signature.
type signature [64]byte

func (s signature) String() string { return fmt.Sprintf("0x%x", s[:]) }

// validityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type validityAttestationValues interface {
	implicit | explicit
}

type validityAttestation struct {
	inner any
}

func setvalidityAttestation[Value validityAttestationValues](mvdt *validityAttestation, value Value) {
	mvdt.inner = value
}

func (mvdt *validityAttestation) SetValue(value any) (err error) {
	switch value := value.(type) {
	case implicit:
		setvalidityAttestation(mvdt, value)
		return

	case explicit:
		setvalidityAttestation(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt validityAttestation) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case implicit:
		return 1, mvdt.inner, nil

	case explicit:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt validityAttestation) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt validityAttestation) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return *new(implicit), nil

	case 2:
		return *new(explicit), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// implicit is for implicit attestation.
type implicit validatorSignature //skipcq

func (i implicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("implicit(%s)", validatorSignature(i))
}

// explicit is for explicit attestation.
type explicit validatorSignature //skipcq

func (e explicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("explicit(%s)", validatorSignature(e))
}

// newValidityAttestation creates a ValidityAttestation varying data type.
func newValidityAttestation() validityAttestation { //skipcq
	return validityAttestation{}
}

// disputeStatement is a statement about a candidate, to be used within the dispute
// resolution process. Statements are either in favour of the candidate's validity
// or against it.
type disputeStatementValues interface {
	validDisputeStatementKind | invalidDisputeStatementKind
}

type disputeStatement struct {
	inner any
}

func setdisputeStatement[Value disputeStatementValues](mvdt *disputeStatement, value Value) {
	mvdt.inner = value
}

func (mvdt *disputeStatement) SetValue(value any) (err error) {
	switch value := value.(type) {
	case validDisputeStatementKind:
		setdisputeStatement(mvdt, value)
		return

	case invalidDisputeStatementKind:
		setdisputeStatement(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt disputeStatement) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case validDisputeStatementKind:
		return 0, mvdt.inner, nil

	case invalidDisputeStatementKind:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt disputeStatement) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt disputeStatement) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(validDisputeStatementKind), nil

	case 1:
		return *new(invalidDisputeStatementKind), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// validDisputeStatementKind is a kind of statements of validity on a candidate.
type validDisputeStatementKind struct {
	inner any
}
type validDisputeStatementKindValues interface {
	explicitValidDisputeStatementKind | backingSeconded | backingValid | approvalChecking
}

func setvalidDisputeStatementKind[Value validDisputeStatementKindValues](mvdt *validDisputeStatementKind, value Value) {
	mvdt.inner = value
}

func (mvdt *validDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case explicitValidDisputeStatementKind:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case backingSeconded:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case backingValid:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case approvalChecking:
		setvalidDisputeStatementKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt validDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case explicitValidDisputeStatementKind:
		return 0, mvdt.inner, nil

	case backingSeconded:
		return 1, mvdt.inner, nil

	case backingValid:
		return 2, mvdt.inner, nil

	case approvalChecking:
		return 3, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt validDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt validDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(explicitValidDisputeStatementKind), nil

	case 1:
		return *new(backingSeconded), nil

	case 2:
		return *new(backingValid), nil

	case 3:
		return *new(approvalChecking), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitValidDisputeStatementKind struct{} //skipcq

func (explicitValidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "explicit valid dispute statement kind"
}

// backingSeconded is a seconded statement on a candidate from the backing phase.
type backingSeconded common.Hash //skipcq

func (b backingSeconded) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingSeconded(%s)", common.Hash(b))
}

// backingValid is a valid statement on a candidate from the backing phase.
type backingValid common.Hash //skipcq

func (b backingValid) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingValid(%s)", common.Hash(b))
}

// approvalChecking is an approval vote from the approval checking phase.
type approvalChecking struct{} //skipcq

func (approvalChecking) String() string { return "approval checking" }

// invalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type invalidDisputeStatementKindValues interface {
	explicitInvalidDisputeStatementKind
}

type invalidDisputeStatementKind struct {
	inner any
}

func setinvalidDisputeStatementKind[Value invalidDisputeStatementKindValues](
	mvdt *invalidDisputeStatementKind, value Value,
) {
	mvdt.inner = value
}

func (mvdt *invalidDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case explicitInvalidDisputeStatementKind:
		setinvalidDisputeStatementKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt invalidDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case explicitInvalidDisputeStatementKind:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt invalidDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}
func (mvdt invalidDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(explicitInvalidDisputeStatementKind), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func (invalidDisputeStatementKind) String() string { //skipcq
	return "invalid dispute statement kind"
}

// explicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitInvalidDisputeStatementKind struct{} //skipcq

func (explicitInvalidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "explicit invalid dispute statement kind"
}

// newDisputeStatement create a new DisputeStatement varying data type.
func newDisputeStatement() disputeStatement { //skipcq
	return disputeStatement{}
}

// collatorID is the collator's relay-chain account ID
type collatorID sr25519.PublicKey

// collatorSignature is the signature on a candidate's block data signed by a collator.
type collatorSignature signature

// validationCodeHash is the blake2-256 hash of the validation code bytes.
type validationCodeHash common.Hash

// candidateDescriptor is a unique descriptor of the candidate receipt.
type candidateDescriptor struct {
	// The ID of the para this is a candidate for.
	ParaID uint32 `scale:"1"`

	// RelayParent is the hash of the relay-chain block this should be executed in
	// the context of.
	// NOTE: the fact that the hash includes this value means that code depends
	// on this for deduplication. Removing this field is likely to break things.
	RelayParent common.Hash `scale:"2"`

	// Collator is the collator's relay-chain account ID
	Collator collatorID `scale:"3"`

	// PersistedValidationDataHash is the blake2-256 hash of the persisted validation data. This is extra data derived from
	// relay-chain state which may vary based on bitfields included before the candidate.
	// Thus it cannot be derived entirely from the relay-parent.
	PersistedValidationDataHash common.Hash `scale:"4"`

	// PovHash is the hash of the `pov-block`.
	PovHash common.Hash `scale:"5"`
	// ErasureRoot is the root of a block's erasure encoding Merkle tree.
	ErasureRoot common.Hash `scale:"6"`

	// Signature on blake2-256 of components of this receipt:
	// The parachain index, the relay parent, the validation data hash, and the `pov_hash`.
	Signature collatorSignature `scale:"7"`

	// ParaHead is the hash of the para header that is being generated by this candidate.
	ParaHead common.Hash `scale:"8"`
	// ValidationCodeHash is the blake2-256 hash of the validation code bytes.
	ValidationCodeHash validationCodeHash `scale:"9"`
}

// upwardMessage is a message from a parachain to its Relay Chain.
type upwardMessage []byte

// outboundHrmpMessage is an HRMP message seen from the perspective of a sender.
type outboundHrmpMessage struct {
	Recipient uint32 `scale:"1"`
	Data      []byte `scale:"2"`
}

// validationCode is Parachain validation code.
type validationCode []byte

// headData is Parachain head data included in the chain.
type headData []byte

// candidateCommitments are Commitments made in a `CandidateReceipt`. Many of these are outputs of validation.
type candidateCommitments struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []upwardMessage `scale:"1"`
	// Horizontal messages sent by the parachain.
	HorizontalMessages []outboundHrmpMessage `scale:"2"`
	// New validation code.
	NewValidationCode *validationCode `scale:"3"`
	// The head-data produced as a result of execution.
	HeadData headData `scale:"4"`
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32 `scale:"5"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"6"`
}

// committedCandidateReceipt is a candidate-receipt with commitments directly included.
type committedCandidateReceipt struct {
	Descriptor  candidateDescriptor  `scale:"1"`
	Commitments candidateCommitments `scale:"2"`
}

// uncheckedSignedAvailabilityBitfield is a set of unchecked signed availability bitfields.
// Should be sorted by validator index, ascending.
type uncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload []byte `scale:"1"`
	// The index of the validator signing this statement.
	ValidatorIndex uint32 `scale:"2"`
	// The signature by the validator of the signed payload.
	Signature signature `scale:"3"`
}

// backedCandidate is a backed (or backable, depending on context) candidate.
type backedCandidate struct {
	// The candidate referred to.
	Candidate committedCandidateReceipt `scale:"1"`
	// The validity votes themselves, expressed as signatures.
	ValidityVotes []validityAttestation `scale:"2"`
	// The indices of the validators within the group, expressed as a bitfield.
	ValidatorIndices []byte `scale:"3"`
}

// multiDisputeStatementSet is a set of dispute statements.
type multiDisputeStatementSet []disputeStatementSet

// validatorIndex is the index of the validator.
type validatorIndex uint32

// validatorSignature is the signature with which parachain validators sign blocks.
type validatorSignature signature

func (v validatorSignature) String() string { return signature(v).String() }

// statement about the candidate.
// Used as translation of `Vec<(DisputeStatement, ValidatorIndex, ValidatorSignature)>` from rust to go
type statement struct {
	ValidatorIndex     validatorIndex
	ValidatorSignature validatorSignature
	DisputeStatement   disputeStatement
}

// disputeStatementSet is a set of statements about a specific candidate.
type disputeStatementSet struct {
	// The candidate referenced by this set.
	CandidateHash common.Hash `scale:"1"`
	// The session index of the candidate.
	Session uint32 `scale:"2"`
	// Statements about the candidate.
	Statements []statement `scale:"3"`
}

// ParachainInherentData is parachains inherent-data passed into the runtime by a block author.
type ParachainInherentData struct {
	// Signed bitfields by validators about availability.
	Bitfields []uncheckedSignedAvailabilityBitfield `scale:"1"`
	// Backed candidates for inclusion in the block.
	BackedCandidates []backedCandidate `scale:"2"`
	// Sets of dispute votes for inclusion,
	Disputes multiDisputeStatementSet `scale:"3"`
	// The parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}
