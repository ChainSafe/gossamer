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

// Signature could be one of Ed25519 Signature, Sr25519 Signature or ECDSA/SECP256k1 Signature.
type Signature [64]byte

func (s Signature) String() string { return fmt.Sprintf("0x%x", s[:]) }

// ValidityAttestation is an Implicit or Explicit attestation to the validity of a parachain
// candidate.
type ValidityAttestation scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *ValidityAttestation) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*va = ValidityAttestation(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (va *ValidityAttestation) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*va)
	return vdt.Value()
}

// Implicit is for Implicit attestation.
type Implicit ValidatorSignature //skipcq

// Index returns VDT index
func (Implicit) Index() uint { //skipcq
	return 1
}

func (i Implicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("Implicit(%s)", ValidatorSignature(i))
}

// Explicit is for Explicit attestation.
type Explicit ValidatorSignature //skipcq

// Index returns VDT index
func (Explicit) Index() uint { //skipcq
	return 2
}

func (e Explicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("Explicit(%s)", ValidatorSignature(e))
}

func (va *ValidityAttestation) Signature() (Signature, error) {
	vdt := scale.VaryingDataType(*va)
	val, err := vdt.Value()
	if err != nil {
		return Signature{}, err
	}

	switch v := val.(type) {
	case Implicit:
		return Signature(v), nil
	case Explicit:
		return Signature(v), nil
	default:
		return Signature{}, fmt.Errorf("invalid validity attestation type")
	}
}

// newValidityAttestation creates a ValidityAttestation varying data type.
func newValidityAttestation() ValidityAttestation { //skipcq
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

// New will enable scale to create new instance when needed
func (d DisputeStatement) New() DisputeStatement {
	return NewDisputeStatement()
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (d *DisputeStatement) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*d)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*d = DisputeStatement(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (d *DisputeStatement) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*d)
	return vdt.Value()
}

// IsValid returns true if the DisputeStatement is valid.
func (d DisputeStatement) IsValid() (bool, error) {
	vdt := scale.VaryingDataType(d)
	val, err := vdt.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatement vdt: %w", err)
	}

	_, ok := val.(ValidDisputeStatementKind)
	return ok, nil
}

// ValidDisputeStatementKind is a kind of statements of validity on a candidate.
type ValidDisputeStatementKind scale.VaryingDataType //skipcq

// Index returns VDT index
func (ValidDisputeStatementKind) Index() uint { //skipcq
	return 0
}

func (ValidDisputeStatementKind) String() string { //skipcq
	return "valid dispute statement kind"
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *ValidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*v)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*v = ValidDisputeStatementKind(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (v *ValidDisputeStatementKind) Value() (scale.VaryingDataTypeValue, error) { //skipcq
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// ExplicitValidDisputeStatementKind is an Explicit statement issued as part of a dispute.
type ExplicitValidDisputeStatementKind struct{} //skipcq

// Index returns VDT index
func (ExplicitValidDisputeStatementKind) Index() uint { //skipcq
	return 0
}

func (ExplicitValidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "Explicit valid dispute statement kind"
}

// BackingSeconded is a seconded statement on a candidate from the backing phase.
type BackingSeconded common.Hash //skipcq

// Index returns VDT index
func (BackingSeconded) Index() uint { //skipcq
	return 1
}

func (b BackingSeconded) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("BackingSeconded(%s)", common.Hash(b))
}

// BackingValid is a valid statement on a candidate from the backing phase.
type BackingValid common.Hash //skipcq

// Index returns VDT index
func (BackingValid) Index() uint { //skipcq
	return 2
}

func (b BackingValid) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("BackingValid(%s)", common.Hash(b))
}

// ApprovalChecking is an approval vote from the approval checking phase.
type ApprovalChecking struct{} //skipcq

// Index returns VDT index
func (ApprovalChecking) Index() uint { //skipcq
	return 3
}

func (ApprovalChecking) String() string { return "approval checking" }

// InvalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type InvalidDisputeStatementKind scale.VaryingDataType //skipcq

// Index returns VDT index
func (InvalidDisputeStatementKind) Index() uint { //skipcq
	return 1
}

func (InvalidDisputeStatementKind) String() string { //skipcq
	return "invalid dispute statement kind"
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (in *InvalidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*in)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*in = InvalidDisputeStatementKind(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (in *InvalidDisputeStatementKind) Value() (scale.VaryingDataTypeValue, error) { //skipcq
	vdt := scale.VaryingDataType(*in)
	return vdt.Value()
}

// ExplicitInvalidDisputeStatementKind is an Explicit statement issued as part of a dispute.
type ExplicitInvalidDisputeStatementKind struct{} //skipcq

// Index returns VDT index
func (ExplicitInvalidDisputeStatementKind) Index() uint { //skipcq
	return 0
}

func (ExplicitInvalidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "Explicit invalid dispute statement kind"
}

// NewValidDisputeStatementKind create a new DisputeStatementKind varying data type.
func NewValidDisputeStatementKind() scale.VaryingDataType {
	vdsKind, err := scale.NewVaryingDataType(
		ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
	if err != nil {
		panic(err)
	}

	return vdsKind
}

// NewInvalidDisputeStatementKind create a new DisputeStatementKind varying data type.
func NewInvalidDisputeStatementKind() scale.VaryingDataType {
	idsKind, err := scale.NewVaryingDataType(ExplicitInvalidDisputeStatementKind{})
	if err != nil {
		panic(err)
	}

	return idsKind
}

// NewDisputeStatement create a new DisputeStatement varying data type.
func NewDisputeStatement() DisputeStatement { //skipcq
	idsKind, err := scale.NewVaryingDataType(ExplicitInvalidDisputeStatementKind{})
	if err != nil {
		panic(err)
	}

	vdsKind, err := scale.NewVaryingDataType(
		ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
	if err != nil {
		panic(err)
	}

	vdt, err := scale.NewVaryingDataType(
		ValidDisputeStatementKind(vdsKind), InvalidDisputeStatementKind(idsKind))
	if err != nil {
		panic(err)
	}

	return DisputeStatement(vdt)
}

//// NewInvalidDisputeStatement create a new DisputeStatement varying data type.
//func NewInvalidDisputeStatement() DisputeStatement { //skipcq
//	disputeStatement := NewDisputeStatement()
//
//	idsKind, err := scale.NewVaryingDataType(ExplicitInvalidDisputeStatementKind{})
//	if err != nil {
//		panic(err)
//	}
//
//	err = disputeStatement.Set(InvalidDisputeStatementKind(idsKind))
//	if err != nil {
//		panic(err)
//	}
//
//	return disputeStatement
//}
//
//// NewValidDisputeStatement create a new DisputeStatement varying data type.
//func NewValidDisputeStatement() DisputeStatement { //skipcq
//	disputeStatement := NewDisputeStatement()
//
//	vdsKind, err := scale.NewVaryingDataType(
//		ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
//	if err != nil {
//		panic(err)
//	}
//
//	err = disputeStatement.Set(ValidDisputeStatementKind(vdsKind))
//	if err != nil {
//		panic(err)
//	}
//
//	return disputeStatement
//}

// collatorID is the collator's relay-chain account ID
type collatorID sr25519.PublicKey

// collatorSignature is the Signature on a candidate's block data signed by a collator.
type collatorSignature Signature

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
	// The Signature by the validator of the signed payload.
	Signature Signature `scale:"3"`
}

// backedCandidate is a backed (or backable, depending on context) candidate.
type backedCandidate struct {
	// The candidate referred to.
	Candidate committedCandidateReceipt `scale:"1"`
	// The validity votes themselves, expressed as signatures.
	ValidityVotes []ValidityAttestation `scale:"2"`
	// The indices of the validators within the group, expressed as a bitfield.
	ValidatorIndices []byte `scale:"3"`
}

// MultiDisputeStatementSet is a set of dispute statements.
type MultiDisputeStatementSet []disputeStatementSet

// validatorIndex is the index of the validator.
type validatorIndex uint32

// ValidatorSignature is the Signature with which parachain validators sign blocks.
type ValidatorSignature Signature

func (v ValidatorSignature) String() string { return Signature(v).String() }

// statement about the candidate.
// Used as translation of `Vec<(DisputeStatement, ValidatorIndex, ValidatorSignature)>` from rust to go
type statement struct {
	ValidatorIndex     validatorIndex
	ValidatorSignature ValidatorSignature
	DisputeStatement   DisputeStatement
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
	Disputes MultiDisputeStatementSet `scale:"3"`
	// The parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}
