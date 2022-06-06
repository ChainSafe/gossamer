package babe

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type ValidityAttestation scale.VaryingDataType

// Implicit is for implicit attestation.
type Implicit ValidatorSignature

// Index Returns VDT index
func (im Implicit) Index() uint {
	return 1
}

// Explicit is for explicit attestation.
type Explicit ValidatorSignature

// Index Returns VDT index
func (ex Explicit) Index() uint {
	return 2
}

// GetValidityAttestation returns a ValidityAttestation varying data type.
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

// Valid is for a valid statement
type Valid ValidDisputeStatementKind

// Index Returns VDT index
func (v Valid) Index() uint {
	return 0
}

// Invalid is for an invalid statement
type Invalid InvalidDisputeStatementKind

// Index Returns VDT index
func (in Invalid) Index() uint {
	return 1
}

// ValidDisputeStatementKind is a kind of statements of validity on a candidate.
type ValidDisputeStatementKind scale.VaryingDataType

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type ExplicitValidDisputeStatementKind struct{}

// Index Returns VDT index
func (ex ExplicitValidDisputeStatementKind) Index() uint {
	return 0
}

// BackingSeconded is a seconded statement on a candidate from the backing phase.
type BackingSeconded common.Hash

// Index Returns VDT index
func (bs BackingSeconded) Index() uint {
	return 1
}

// BackingValid is a valid statement on a candidate from the backing phase.
type BackingValid common.Hash

// Index Returns VDT index
func (bv BackingValid) Index() uint {
	return 2
}

// ApprovalChecking is an approval vote from the approval checking phase.
type ApprovalChecking struct{}

// Index Returns VDT index
func (ac ApprovalChecking) Index() uint {
	return 3
}

// InvalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type InvalidDisputeStatementKind scale.VaryingDataType

// ExplicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type ExplicitInvalidDisputeStatementKind struct{}

// Index Returns VDT index
func (ex ExplicitInvalidDisputeStatementKind) Index() uint {
	return 0
}

// GetDisputeStatement returns a GetDisputeStatement varying data type.
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

	vdt, err := scale.NewVaryingDataType(validDisputeStatementKind.Value(), invalidDisputeStatementKind.Value())
	if err != nil {
		panic(err)
	}

	return DisputeStatement(vdt)
}

// CandidateDescriptor is a unique descriptor of the candidate receipt.
type CandidateDescriptor struct {
	// The ID of the para this is a candidate for.
	ParaID uint32
	// The hash of the relay-chain block this should be executed in
	// the context of.
	// NOTE: the fact that the hash includes this value means that code depends
	// on this for deduplication. Removing this field is likely to break things.
	RelayParent common.Hash
	// The collator's relay-chain account ID
	Collator []byte // CollatorId
	// Signature on blake2-256 of components of this receipt:
	// The para ID, the relay parent, and the `pov_hash`.
	Signature []byte // CollatorSignature
	// The hash of the `pov-block`.
	PovHash common.Hash
}

// UpwardMessage is a message from a parachain to its Relay Chain.
type UpwardMessage []byte

// OutboundHrmpMessage is an HRMP message seen from the perspective of a sender.
type OutboundHrmpMessage struct {
	Recipient uint32
	Data      []byte
}

// ValidationCode is Parachain validation code.
type ValidationCode []byte

// HeadData is Parachain head data included in the chain.
type HeadData []byte

// CandidateCommitments are Commitments made in a `CandidateReceipt`. Many of these are outputs of validation.
type CandidateCommitments struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []UpwardMessage
	// Horizontal messages sent by the parachain.
	HorizontalMessages []OutboundHrmpMessage
	// New validation code.
	NewValidationCode *ValidationCode
	// The head-data produced as a result of execution.
	HeadData HeadData
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32
}

// CommittedCandidateReceipt is a candidate-receipt with commitments directly included.
type CommittedCandidateReceipt struct {
	Descriptor  *CandidateDescriptor
	Commitments *CandidateCommitments
}

// UncheckedSignedAvailabilityBitfield is a set of unchecked signed availability bitfields.
// Should be sorted by validator index, ascending.
type UncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload []byte
	// The index of the validator signing this statement.
	ValidatorIndex uint32
	/// The signature by the validator of the signed payload.
	Signature []byte
}

// BackedCandidate is a backed (or backable, depending on context) candidate.
type BackedCandidate struct {
	// The candidate referred to.
	Candidate *CommittedCandidateReceipt
	// The validity votes themselves, expressed as signatures.
	ValidityVotes []*ValidityAttestation
	// The indices of the validators within the group, expressed as a bitfield.
	ValidatorIndices []byte
}

// MultiDisputeStatementSet is a set of dispute statements.
type MultiDisputeStatementSet []DisputeStatementSet

// ValidatorIndex is the index of the validator.
type ValidatorIndex uint32

// ValidatorSignature is the signature with which parachain validators sign blocks.
type ValidatorSignature []byte

// Statement about the candidate.
type Statement struct {
	ValidatorIndex     ValidatorIndex
	ValidatorSignature ValidatorSignature
	DisputeStatement   DisputeStatement
}

// DisputeStatementSet is a set of statements about a specific candidate.
type DisputeStatementSet struct {
	// The candidate referenced by this set.
	CandidateHash common.Hash
	// The session index of the candidate.
	Session uint32
	// Statements about the candidate.
	Statements []Statement
}

// ParachainInherentData is parachains inherent-data passed into the runtime by a block author.
type ParachainInherentData struct {
	// Signed bitfields by validators about availability.
	Bitfields []UncheckedSignedAvailabilityBitfield
	// Backed candidates for inclusion in the block.
	BackedCandidates []BackedCandidate
	// Sets of dispute votes for inclusion,
	Disputes MultiDisputeStatementSet // []DisputeStatementSet
	// The parent block header. Used for checking state proofs.
	ParentHeader *types.Header
}
