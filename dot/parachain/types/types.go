// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// The primary purpose of this package is to put types being used by other packages to avoid cyclic
// dependencies.

// NOTE: https://github.com/ChainSafe/gossamer/pull/3297#discussion_r1214740051

// ValidatorIndex Index of the validator. Used as a lightweight replacement of the `ValidatorId` when appropriate
type ValidatorIndex uint32

// ValidatorID The public key of a validator.
type ValidatorID [sr25519.PublicKeyLength]byte

// BlockNumber The block number type.
type BlockNumber uint32

// GroupRotationInfo A helper data-type for tracking validator-group rotations.
type GroupRotationInfo struct {
	// SessionStartBlock is the block number at which the session started
	SessionStartBlock BlockNumber `scale:"1"`
	// GroupRotationFrequency indicates how often groups rotate. 0 means never.
	GroupRotationFrequency BlockNumber `scale:"2"`
	// Now indicates the current block number.
	Now BlockNumber `scale:"3"`
}

func (gri GroupRotationInfo) CoreForGroup(groupIndex GroupIndex, cores uint8) CoreIndex {
	// TODO https://github.com/paritytech/polkadot-sdk/blob/aa68ea58f389c2aa4eefab4bf7bc7b787dd56580/polkadot/primitives/src/v6/mod.rs#L877 //nolint
	return CoreIndex{}
}

// ValidatorGroups represents the validator groups
type ValidatorGroups struct {
	// Validators is an array of validator set Ids
	Validators [][]ValidatorIndex `scale:"1"`
	// GroupRotationInfo is the group rotation info
	GroupRotationInfo GroupRotationInfo `scale:"2"`
}

// ParaID Unique identifier of a parachain.
type ParaID uint32

// GroupIndex The unique (during session) index of a validator group.
type GroupIndex uint32

// CollatorID represents the public key of a collator
type CollatorID [sr25519.PublicKeyLength]byte

// CollatorSignature is the signature on a candidate's block data signed by a collator.
type CollatorSignature [sr25519.SignatureLength]byte

// ValidationCodeHash is the blake2-256 hash of the validation code bytes.
type ValidationCodeHash common.Hash

// CandidateDescriptor is a unique descriptor of the candidate receipt.
type CandidateDescriptor struct {
	// The ID of the para this is a candidate for.
	ParaID uint32 `scale:"1"`

	// RelayParent is the hash of the relay-chain block this should be executed in
	// the context of.
	RelayParent common.Hash `scale:"2"`

	// Collator is the collator's sr25519 public key.
	Collator CollatorID `scale:"3"`

	// PersistedValidationDataHash is the blake2-256 hash of the persisted validation data. This is extra data derived from
	// relay-chain state which may vary based on bitfields included before the candidate.
	// Thus, it cannot be derived entirely from the relay-parent.
	PersistedValidationDataHash common.Hash `scale:"4"`

	// PovHash is the hash of the `pov-block`.
	PovHash common.Hash `scale:"5"`
	// ErasureRoot is the root of a block's erasure encoding Merkle tree.
	ErasureRoot common.Hash `scale:"6"`

	// Signature on blake2-256 of components of this receipt:
	// The parachain index, the relay parent, the validation data hash, and the `pov_hash`.
	Signature CollatorSignature `scale:"7"`

	// ParaHead is the hash of the para header that is being generated by this candidate.
	ParaHead common.Hash `scale:"8"`
	// ValidationCodeHash is the blake2-256 hash of the validation code bytes.
	ValidationCodeHash ValidationCodeHash `scale:"9"`
}

func (cd CandidateDescriptor) CreateSignaturePayload() ([]byte, error) {
	var payload [132]byte
	copy(payload[0:32], cd.RelayParent.ToBytes())

	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(cd.ParaID)
	if err != nil {
		return nil, fmt.Errorf("encoding parachain id: %w", err)
	}
	if len(buffer.Bytes()) != 4 {
		return nil, fmt.Errorf("invalid length of encoded parachain id")
	}
	copy(payload[32:36], buffer.Bytes())
	copy(payload[36:68], cd.PersistedValidationDataHash.ToBytes())
	copy(payload[68:100], cd.PovHash.ToBytes())
	copy(payload[100:132], common.Hash(cd.ValidationCodeHash).ToBytes())

	return payload[:], nil
}

func (cd CandidateDescriptor) CheckCollatorSignature() error {
	payload, err := cd.CreateSignaturePayload()
	if err != nil {
		return fmt.Errorf("creating signature payload: %w", err)
	}

	return sr25519.VerifySignature(cd.Collator[:], cd.Signature[:], payload)
}

// OccupiedCore Information about a core which is currently occupied.
type OccupiedCore struct {
	// NOTE: this has no ParaId as it can be deduced from the candidate descriptor.
	// If this core is freed by availability, this is the assignment that is next up on this
	// core, if any. nil if there is nothing queued for this core.
	NextUpOnAvailable *ScheduledCore `scale:"1"`
	// The relay-chain block number this began occupying the core at.
	OccupiedSince BlockNumber `scale:"2"`
	// The relay-chain block this will time-out at, if any.
	TimeoutAt BlockNumber `scale:"3"`
	// If this core is freed by being timed-out, this is the assignment that is next up on this
	// core. nil if there is nothing queued for this core or there is no possibility of timing
	// out.
	NextUpOnTimeOut *ScheduledCore `scale:"4"`
	// A bitfield with 1 bit for each validator in the set. `1` bits mean that the corresponding
	// validators has attested to availability on-chain. A 2/3+ majority of `1` bits means that
	// this will be available.
	Availability scale.BitVec `scale:"5"`
	// The group assigned to distribute availability pieces of this candidate.
	GroupResponsible GroupIndex `scale:"6"`
	// The hash of the candidate occupying the core.
	CandidateHash common.Hash `scale:"7"`
	// The descriptor of the candidate occupying the core.
	CandidateDescriptor CandidateDescriptor `scale:"8"`
}

// Index returns the index
func (OccupiedCore) Index() uint {
	return 0
}

// ScheduledCore Information about a core which is currently occupied.
type ScheduledCore struct {
	// The ID of a para scheduled.
	ParaID ParaID `scale:"1"`
	// The collator required to author the block, if any.
	Collator *CollatorID `scale:"2"`
}

// Index returns the index
func (ScheduledCore) Index() uint {
	return 1
}

// Free Core information about a core which is currently free.
type Free struct{}

// Index returns the index
func (Free) Index() uint {
	return 2
}

// CoreState represents the state of a particular availability core.
type CoreState scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *CoreState) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*va = CoreState(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (va *CoreState) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*va)
	return vdt.Value()
}

// NewCoreStateVDT returns a new CoreState VaryingDataType
func NewCoreStateVDT() (scale.VaryingDataType, error) {
	vdt, err := scale.NewVaryingDataType(OccupiedCore{}, ScheduledCore{}, Free{})
	if err != nil {
		return scale.VaryingDataType{}, fmt.Errorf("create varying data type: %w", err)
	}

	return vdt, nil
}

// NewAvailabilityCores returns a new AvailabilityCores
func NewAvailabilityCores() (scale.VaryingDataTypeSlice, error) {
	vdt, err := NewCoreStateVDT()
	if err != nil {
		return scale.VaryingDataTypeSlice{}, fmt.Errorf("create varying data type: %w", err)
	}

	return scale.NewVaryingDataTypeSlice(vdt), nil
}

// UpwardMessage A message from a parachain to its Relay Chain.
type UpwardMessage []byte

// OutboundHrmpMessage is an HRMP message seen from the perspective of a sender.
type OutboundHrmpMessage struct {
	Recipient uint32 `scale:"1"`
	Data      []byte `scale:"2"`
}

// ValidationCode is Parachain validation code.
type ValidationCode []byte

// CandidateCommitments are Commitments made in a `CandidateReceipt`. Many of these are outputs of validation.
type CandidateCommitments struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []UpwardMessage `scale:"1"`
	// Horizontal messages sent by the parachain.
	HorizontalMessages []OutboundHrmpMessage `scale:"2"`
	// New validation code.
	NewValidationCode *ValidationCode `scale:"3"`
	// The head-data produced as a result of execution.
	HeadData HeadData `scale:"4"`
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32 `scale:"5"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"6"`
}

func (cc CandidateCommitments) Hash() common.Hash {
	return common.MustBlake2bHash(scale.MustMarshal(cc))
}

// SessionIndex is a session index.
type SessionIndex uint32

// CommittedCandidateReceipt A candidate-receipt with commitments directly included.
type CommittedCandidateReceipt struct {
	// The candidate descriptor.
	Descriptor CandidateDescriptor `scale:"1"`
	// The commitments made by the parachain.
	Commitments CandidateCommitments `scale:"2"`
}

func (ccr CommittedCandidateReceipt) ToPlain() CandidateReceipt {
	return CandidateReceipt{
		Descriptor:      ccr.Descriptor,
		CommitmentsHash: ccr.Commitments.Hash(),
	}
}

func (c CommittedCandidateReceipt) Hash() (common.Hash, error) {
	return c.ToPlain().Hash()
}

// AssignmentID The public key of a keypair used by a validator for determining assignments
// to approve included parachain candidates.
type AssignmentID [sr25519.PublicKeyLength]byte

// AuthorityDiscoveryID An authority discovery identifier.
type AuthorityDiscoveryID [sr25519.PublicKeyLength]byte

// SessionInfo Information about validator sets of a session.
type SessionInfo struct {
	// All the validators actively participating in parachain consensus.
	// Indices are into the broader validator set.
	ActiveValidatorIndices []ValidatorIndex `scale:"1"`
	// A secure random seed for the session, gathered from BABE.
	RandomSeed [32]byte `scale:"2"`
	// The amount of sessions to keep for disputes.
	DisputePeriod SessionIndex `scale:"3"`
	// Validators in canonical ordering.
	Validators []ValidatorID `scale:"4"`
	// Validators' authority discovery keys for the session in canonical ordering.
	DiscoveryKeys []AuthorityDiscoveryID `scale:"5"`
	// The assignment keys for validators.
	AssignmentKeys []AssignmentID `scale:"6"`
	// Validators in shuffled ordering - these are the validator groups as produced
	// by the `Scheduler` module for the session and are typically referred to by
	// `GroupIndex`.
	ValidatorGroups [][]ValidatorIndex `scale:"7"`
	// The number of availability cores used by the protocol during this session.
	NCores uint32 `scale:"8"`
	// The zeroth delay tranche width.
	ZerothDelayTrancheWidth uint32 `scale:"9"`
	// The number of samples we do of `relay_vrf_modulo`.
	RelayVRFModuloSamples uint32 `scale:"10"`
	// The number of delay tranches in total.
	NDelayTranches uint32 `scale:"11"`
	// How many slots (BABE / SASSAFRAS) must pass before an assignment is considered a
	// no-show.
	NoShowSlots uint32 `scale:"12"`
	// The number of validators needed to approve a block.
	NeededApprovals uint32 `scale:"13"`
}

// DownwardMessage A message sent from the relay-chain down to a parachain.
type DownwardMessage []byte

// InboundDownwardMessage A wrapped version of `DownwardMessage`.
// The difference is that it has attached the block number when the message was sent.
type InboundDownwardMessage struct {
	// The block number at which these messages were put into the downward message queue.
	SentAt BlockNumber `scale:"1"`
	// The actual downward message to processes.
	Message DownwardMessage `scale:"2"`
}

// InboundHrmpMessage An HRMP message seen from the perspective of a recipient.
type InboundHrmpMessage struct {
	// The block number at which this message was sent.
	// Specifically, it is the block number at which the candidate that sends this message was
	// enacted.
	SentAt BlockNumber `scale:"1"`
	// The message payload.
	Data []byte `scale:"2"`
}

// CandidateReceipt A receipt for a parachain candidate.
type CandidateReceipt struct {
	// The candidate descriptor.
	Descriptor CandidateDescriptor `scale:"1"`
	// The candidate event.
	CommitmentsHash common.Hash `scale:"2"`
}

func (cr CandidateReceipt) Hash() (common.Hash, error) {
	bytes, err := scale.Marshal(cr)
	if err != nil {
		return common.Hash{}, fmt.Errorf("marshalling CommittedCandidateReceipt: %w", err)
	}

	return common.Blake2bHash(bytes)
}

// HeadData Parachain head data included in the chain.
type HeadData struct {
	Data []byte `scale:"1"`
}

// CoreIndex The unique (during session) index of a core.
type CoreIndex struct {
	Index uint32 `scale:"1"`
}

// CandidateBacked This candidate receipt was backed in the most recent block.
// This includes the core index the candidate is now occupying.
type CandidateBacked struct {
	CandidateReceipt CandidateReceipt `scale:"1"`
	HeadData         HeadData         `scale:"2"`
	CoreIndex        CoreIndex        `scale:"3"`
	GroupIndex       GroupIndex       `scale:"4"`
}

// Index returns the VaryingDataType Index
func (CandidateBacked) Index() uint {
	return 0
}

// CandidateIncluded This candidate receipt was included and became a parablock at the most recent block.
// This includes the core index the candidate was occupying as well as the group responsible
// for backing the candidate.
type CandidateIncluded struct {
	CandidateReceipt CandidateReceipt `scale:"1"`
	HeadData         HeadData         `scale:"2"`
	CoreIndex        CoreIndex        `scale:"3"`
	GroupIndex       GroupIndex       `scale:"4"`
}

// Index returns the VaryingDataType Index
func (CandidateIncluded) Index() uint {
	return 1
}

// CandidateTimedOut A candidate that timed out.
// / This candidate receipt was not made available in time and timed out.
// / This includes the core index the candidate was occupying.
type CandidateTimedOut struct {
	CandidateReceipt CandidateReceipt `scale:"1"`
	HeadData         HeadData         `scale:"2"`
	CoreIndex        CoreIndex        `scale:"3"`
}

// Index returns the VaryingDataType Index
func (CandidateTimedOut) Index() uint {
	return 2
}

// CandidateEvent A candidate event.
type CandidateEvent scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *CandidateEvent) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*va = CandidateEvent(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (va *CandidateEvent) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*va)
	return vdt.Value()
}

// NewCandidateEventVDT returns a new CandidateEvent VaryingDataType
func NewCandidateEventVDT() (scale.VaryingDataType, error) {
	vdt, err := scale.NewVaryingDataType(CandidateBacked{}, CandidateIncluded{}, CandidateTimedOut{})
	if err != nil {
		return scale.VaryingDataType{}, fmt.Errorf("create varying data type: %w", err)
	}

	return vdt, nil
}

// NewCandidateEvents returns a new CandidateEvents
func NewCandidateEvents() (scale.VaryingDataTypeSlice, error) {
	vdt, err := NewCandidateEventVDT()
	if err != nil {
		return scale.VaryingDataTypeSlice{}, fmt.Errorf("create varying data type: %w", err)
	}

	return scale.NewVaryingDataTypeSlice(vdt), nil
}

// PersistedValidationData should be relatively lightweight primarily because it is constructed
// during inclusion for each candidate and therefore lies on the critical path of inclusion.
type PersistedValidationData struct {
	ParentHead             HeadData    `scale:"1"`
	RelayParentNumber      uint32      `scale:"2"`
	RelayParentStorageRoot common.Hash `scale:"3"`
	MaxPovSize             uint32      `scale:"4"`
}

func (pvd PersistedValidationData) Hash() (common.Hash, error) {
	bytes, err := scale.Marshal(pvd)
	if err != nil {
		return common.Hash{}, fmt.Errorf("marshalling PersistedValidationData: %w", err)
	}

	return common.Blake2bHash(bytes)
}

// OccupiedCoreAssumption is an assumption being made about the state of an occupied core.
type OccupiedCoreAssumption scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (o *OccupiedCoreAssumption) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*o)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
	*o = OccupiedCoreAssumption(vdt)
	return nil
}

// Value will return value from underying VaryingDataType
func (o *OccupiedCoreAssumption) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*o)
	return vdt.Value()
}

// IncludedOccupiedCoreAssumption means the candidate occupying the core was made available and
// included to free the core.
type IncludedOccupiedCoreAssumption struct{}

// Index returns VDT index
func (IncludedOccupiedCoreAssumption) Index() uint {
	return 0
}

func (IncludedOccupiedCoreAssumption) String() string {
	return "Included"
}

// TimedOutOccupiedCoreAssumption means the candidate occupying the core timed out and freed the
// core without advancing the para.
type TimedOutOccupiedCoreAssumption struct{}

// Index returns VDT index
func (TimedOutOccupiedCoreAssumption) Index() uint {
	return 1
}

func (TimedOutOccupiedCoreAssumption) String() string {
	return "TimedOut"
}

// FreeOccupiedCoreAssumption means the core was not occupied to begin with.
type FreeOccupiedCoreAssumption struct{}

// Index returns VDT index
func (FreeOccupiedCoreAssumption) Index() uint {
	return 2
}

func (FreeOccupiedCoreAssumption) String() string {
	return "Free"
}

// NewOccupiedCoreAssumption creates a OccupiedCoreAssumption varying data type.
func NewOccupiedCoreAssumption() OccupiedCoreAssumption {
	vdt := scale.MustNewVaryingDataType(
		IncludedOccupiedCoreAssumption{},
		FreeOccupiedCoreAssumption{},
		TimedOutOccupiedCoreAssumption{})

	return OccupiedCoreAssumption(vdt)
}

// CandidateHash makes it easy to enforce that a hash is a candidate hash on the type level.
type CandidateHash struct {
	Value common.Hash `scale:"1"`
}

func (ch CandidateHash) String() string {
	return ch.Value.String()
}

// PoV represents a Proof-of-Validity block (PoV block) or a parachain block.
// It contains the necessary data for the parachain specific state transition logic.
type PoV struct {
	BlockData BlockData `scale:"1"`
}

// Index returns the index of varying data type
func (PoV) Index() uint {
	return 0
}

// NoSuchPoV indicates that the requested PoV was not found in the store.
type NoSuchPoV struct{}

// Index returns the index of varying data type
func (NoSuchPoV) Index() uint {
	return 1
}

// BlockData represents parachain block data.
// It contains everything required to validate para-block, may contain block and witness data.
type BlockData []byte

// Collation represents a requested collation to be delivered
type Collation struct {
	CandidateReceipt CandidateReceipt `scale:"1"`
	PoV              PoV              `scale:"2"`
}

// ValidatorSignature represents the signature with which parachain validators sign blocks.
type ValidatorSignature Signature

func (v ValidatorSignature) String() string { return Signature(v).String() }

// Signature represents a cryptographic signature.
type Signature [64]byte

func (s Signature) String() string { return fmt.Sprintf("0x%x", s[:]) }

// BackedCandidate is a backed (or backable, depending on context) candidate.
type BackedCandidate struct {
	// The candidate referred to.
	Candidate CommittedCandidateReceipt `scale:"1"`
	// The validity votes themselves, expressed as signatures.
	ValidityVotes []ValidityAttestation `scale:"2"`
	// The indices of the validators within the group, expressed as a bitfield.
	ValidatorIndices scale.BitVec `scale:"3"` // TODO: it's a bitvec in rust, figure out actual type
}

// ProspectiveParachainsMode represents the mode of a relay parent in the context
// of prospective parachains, as defined by the Runtime API version.
type ProspectiveParachainsMode struct {
	// IsEnabled indicates whether prospective parachains are enabled or disabled.
	// - Disabled: When the Runtime API lacks support for `async_backing_params`,
	//   there are no prospective parachains.
	// - Enabled: For v6 runtime API, prospective parachains are enabled.
	// NOTE: MaxCandidateDepth and AllowedAncestryLen should be set only if this is enabled.
	IsEnabled bool

	// MaxCandidateDepth specifies the maximum number of para blocks that can exist
	// between the para head in a relay parent and a new candidate. This limitation
	// helps prevent the construction of arbitrarily long chains and spamming by
	// other validators.
	MaxCandidateDepth uint

	// AllowedAncestryLen determines how many ancestors of a relay parent are allowed
	// to build candidates on top of it.
	AllowedAncestryLen uint
}

// UncheckedSignedAvailabilityBitfield a signed bitfield with signature not yet checked
type UncheckedSignedAvailabilityBitfield struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload scale.BitVec `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}
