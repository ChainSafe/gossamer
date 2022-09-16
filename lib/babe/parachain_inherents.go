// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// signature could be one of Ed25519 signature, Sr25519 signature or ECDSA/SECP256k1 signature.
type Signature [64]byte

// ValidityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type ValidityAttestation scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (validityAttestation *ValidityAttestation) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*validityAttestation)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*validityAttestation = ValidityAttestation(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (validityAttestation *ValidityAttestation) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*validityAttestation)
	return vdt.Value()
}

// Implicit is for implicit attestation.
type Implicit ValidatorSignature

// Index returns VDT index
func (Implicit) Index() uint {
	return 1
}

// Explicit is for explicit attestation.
type Explicit ValidatorSignature

// Index returns VDT index
func (Explicit) Index() uint {
	return 2
}

// NewValidityAttestation creates a ValidityAttestation varying data type.
func NewValidityAttestation() ValidityAttestation {
	vdt, err := scale.NewVaryingDataType(Implicit{}, Explicit{})
	if err != nil {
		panic(err)
	}

	return ValidityAttestation(vdt)
}

// DisputeStatement is a statement about a candidate, to be used within the dispute
// resolution process. Statements are either in favour of the candidate's validity
// or against it.
type DisputeStatement scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (distputedStatement *DisputeStatement) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*distputedStatement)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*distputedStatement = DisputeStatement(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (distputedStatement *DisputeStatement) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*distputedStatement)
	return vdt.Value()
}

// ValidDisputeStatementKind is a kind of statements of validity on a candidate.
type ValidDisputeStatementKind scale.VaryingDataType

// Index returns VDT index
func (ValidDisputeStatementKind) Index() uint {
	return 0
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *ValidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*v = ValidDisputeStatementKind(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (v *ValidDisputeStatementKind) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type ExplicitValidDisputeStatementKind struct{}

// Index returns VDT index
func (ExplicitValidDisputeStatementKind) Index() uint {
	return 0
}

// BackingSeconded is a seconded statement on a candidate from the backing phase.
type BackingSeconded common.Hash

// Index returns VDT index
func (BackingSeconded) Index() uint {
	return 1
}

// BackingValid is a valid statement on a candidate from the backing phase.
type BackingValid common.Hash

// Index returns VDT index
func (BackingValid) Index() uint {
	return 2
}

// ApprovalChecking is an approval vote from the approval checking phase.
type ApprovalChecking struct{}

// Index returns VDT index
func (ApprovalChecking) Index() uint {
	return 3
}

// InvalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type InvalidDisputeStatementKind scale.VaryingDataType

// Index returns VDT index
func (InvalidDisputeStatementKind) Index() uint {
	return 1
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (in *InvalidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*in)
	err = vdt.Set(val)
	if err != nil {
		return
	}
	// store original ParentVDT with VaryingDataType that has been set
	*in = InvalidDisputeStatementKind(vdt)
	return
}

// Value will return value from underying VaryingDataType
func (in *InvalidDisputeStatementKind) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*in)
	return vdt.Value()
}

// ExplicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type ExplicitInvalidDisputeStatementKind struct{}

// Index returns VDT index
func (ExplicitInvalidDisputeStatementKind) Index() uint {
	return 0
}

// NewDisputeStatement create a new DisputeStatement varying data type.
func NewDisputeStatement() DisputeStatement {
	invalidDisputeStatementKind, err := scale.NewVaryingDataType(ExplicitInvalidDisputeStatementKind{})
	if err != nil {
		panic(err)
	}

	validDisputeStatementKind, err := scale.NewVaryingDataType(
		ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
	if err != nil {
		panic(err)
	}

	vdt, err := scale.NewVaryingDataType(
		ValidDisputeStatementKind(validDisputeStatementKind), InvalidDisputeStatementKind(invalidDisputeStatementKind))
	if err != nil {
		panic(err)
	}

	return DisputeStatement(vdt)
}

// collatorID is the collator's relay-chain account ID
type collatorID []byte

// collatorSignature is signature on candidate's block data by a collator.
type collatorSignature Signature

//  validationCodeHash is the blake2-256 hash of the validation code bytes.
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

// HeadData is Parachain head data included in the chain.
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

// UncheckedSignedAvailabilityBitfield is a set of unchecked signed availability bitfields.
// Should be sorted by validator index, ascending.
type UncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload []byte `scale:"1"`
	// The index of the validator signing this statement.
	ValidatorIndex uint32 `scale:"2"`
	/// The signature by the validator of the signed payload.
	Signature Signature `scale:"3"`
	// go does not have phantom data
	// /// This ensures the real payload is tracked at the typesystem level.
	// real_payload: sp_std::marker::PhantomData<RealPayload>,
}

// BackedCandidate is a backed (or backable, depending on context) candidate.
type BackedCandidate struct {
	// The candidate referred to.
	Candidate committedCandidateReceipt `scale:"1"`
	// The validity votes themselves, expressed as signatures.
	ValidityVotes []ValidityAttestation `scale:"2"`
	// The indices of the validators within the group, expressed as a bitfield.
	ValidatorIndices []byte `scale:"3"`
}

// MultiDisputeStatementSet is a set of dispute statements.
type MultiDisputeStatementSet []DisputeStatementSet

// ValidatorIndex is the index of the validator.
type ValidatorIndex uint32

// ValidatorSignature is the signature with which parachain validators sign blocks.
type ValidatorSignature Signature

// Statement about the candidate.
// Used as translation of `Vec<(DisputeStatement, ValidatorIndex, ValidatorSignature)>` from rust to go
type Statement struct {
	ValidatorIndex     ValidatorIndex
	ValidatorSignature ValidatorSignature
	DisputeStatement   DisputeStatement
}

// DisputeStatementSet is a set of statements about a specific candidate.
type DisputeStatementSet struct {
	// The candidate referenced by this set.
	CandidateHash common.Hash `scale:"1"`
	// The session index of the candidate.
	Session uint32 `scale:"2"`
	// Statements about the candidate.
	Statements []Statement `scale:"3"`
}

// ParachainInherentData is parachains inherent-data passed into the runtime by a block author.
type ParachainInherentData struct {
	// Signed bitfields by validators about availability.
	Bitfields []UncheckedSignedAvailabilityBitfield `scale:"1"`
	// Backed candidates for inclusion in the block.
	BackedCandidates []BackedCandidate `scale:"2"`
	// Sets of dispute votes for inclusion,
	Disputes MultiDisputeStatementSet `scale:"3"` // []DisputeStatementSet
	// The parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}
