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
type signature [64]byte

// validityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type validityAttestation scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *validityAttestation) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*va = validityAttestation(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (va *validityAttestation) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*va)
	return vdt.Value()
}

// implicit is for implicit attestation.
type implicit validatorSignature

// Index returns VDT index
func (implicit) Index() uint {
	return 1
}

// explicit is for explicit attestation.
type explicit validatorSignature

// Index returns VDT index
func (explicit) Index() uint {
	return 2
}

// newValidityAttestation creates a ValidityAttestation varying data type.
func newValidityAttestation() validityAttestation {
	vdt, err := scale.NewVaryingDataType(implicit{}, explicit{})
	if err != nil {
		panic(err)
	}

	return validityAttestation(vdt)
}

// disputeStatement is a statement about a candidate, to be used within the dispute
// resolution process. Statements are either in favour of the candidate's validity
// or against it.
type disputeStatement scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (d *disputeStatement) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*d)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*d = disputeStatement(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (distputedStatement *disputeStatement) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*distputedStatement)
	return vdt.Value()
}

// validDisputeStatementKind is a kind of statements of validity on a candidate.
type validDisputeStatementKind scale.VaryingDataType

// Index returns VDT index
func (validDisputeStatementKind) Index() uint {
	return 0
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *validDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*v = validDisputeStatementKind(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (v *validDisputeStatementKind) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitValidDisputeStatementKind struct{}

// Index returns VDT index
func (explicitValidDisputeStatementKind) Index() uint {
	return 0
}

// backingSeconded is a seconded statement on a candidate from the backing phase.
type backingSeconded common.Hash

// Index returns VDT index
func (backingSeconded) Index() uint {
	return 1
}

// backingValid is a valid statement on a candidate from the backing phase.
type backingValid common.Hash

// Index returns VDT index
func (backingValid) Index() uint {
	return 2
}

// approvalChecking is an approval vote from the approval checking phase.
type approvalChecking struct{}

// Index returns VDT index
func (approvalChecking) Index() uint {
	return 3
}

// invalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type invalidDisputeStatementKind scale.VaryingDataType

// Index returns VDT index
func (invalidDisputeStatementKind) Index() uint {
	return 1
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (in *invalidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*in)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*in = invalidDisputeStatementKind(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (in *invalidDisputeStatementKind) Value() (val scale.VaryingDataTypeValue) {
	vdt := scale.VaryingDataType(*in)
	return vdt.Value()
}

// explicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitInvalidDisputeStatementKind struct{}

// Index returns VDT index
func (explicitInvalidDisputeStatementKind) Index() uint {
	return 0
}

// newDisputeStatement create a new DisputeStatement varying data type.
func newDisputeStatement() disputeStatement {
	idsKind, err := scale.NewVaryingDataType(explicitInvalidDisputeStatementKind{})
	if err != nil {
		panic(err)
	}

	vdsKind, err := scale.NewVaryingDataType(
		explicitValidDisputeStatementKind{}, backingSeconded{}, backingValid{}, approvalChecking{})
	if err != nil {
		panic(err)
	}

	vdt, err := scale.NewVaryingDataType(
		validDisputeStatementKind(vdsKind), invalidDisputeStatementKind(idsKind))
	if err != nil {
		panic(err)
	}

	return disputeStatement(vdt)
}

// collatorID is the collator's relay-chain account ID
type collatorID []byte

// CollatorSignature is the signature on a candidate's block data signed by a collator.
type collatorSignature signature

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

// uncheckedSignedAvailabilityBitfield is a set of unchecked signed availability bitfields.
// Should be sorted by validator index, ascending.
type uncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload []byte `scale:"1"`
	// The index of the validator signing this statement.
	ValidatorIndex uint32 `scale:"2"`
	/// The signature by the validator of the signed payload.
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
	Disputes multiDisputeStatementSet `scale:"3"` // []DisputeStatementSet
	// The parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}
