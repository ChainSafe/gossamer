// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// GroupRotationInfo A helper data-type for tracking validator-group rotations.
type GroupRotationInfo struct {
	// SessionStartBlock is the block number at which the session started
	SessionStartBlock uint32 `scale:"1"`
	// GroupRotationFrequency indicates how often groups rotate. 0 means never.
	GroupRotationFrequency uint32 `scale:"2"`
	// Now indicates the current block number.
	Now uint32 `scale:"3"`
}

// ValidatorGroups represents the validator groups
type ValidatorGroups struct {
	// Validators is an array of validator set Ids
	Validators [][]types.ValidatorIndex `scale:"1"`
	// GroupRotationInfo is the group rotation info
	GroupRotationInfo GroupRotationInfo `scale:"2"`
}

// ParaID The ID of a parachain.
type ParaID uint32

// GroupIndex The unique (during session) index of a validator group.
type GroupIndex uint32

// CollatorID represents the public key of a collator
type CollatorID [sr25519.PublicKeyLength]byte

// Collator represents a collator
type Collator struct {
	Key crypto.PublicKey
}

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
	// NOTE: the fact that the hash includes this value means that code depends
	// on this for deduplication. Removing this field is likely to break things.
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
	// this is basically sr25519::Signature
	Signature CollatorSignature `scale:"7"`

	// ParaHead is the hash of the para header that is being generated by this candidate.
	ParaHead common.Hash `scale:"8"`
	// ValidationCodeHash is the blake2-256 hash of the validation code bytes.
	ValidationCodeHash ValidationCodeHash `scale:"9"`
}

// OccupiedCore Information about a core which is currently occupied.
type OccupiedCore struct {
	// NOTE: this has no ParaId as it can be deduced from the candidate descriptor.
	// If this core is freed by availability, this is the assignment that is next up on this
	// core, if any. None if there is nothing queued for this core.
	NextUpOnAvailable *ScheduledCore `scale:"1"`
	// The relay-chain block number this began occupying the core at.
	OccupiedSince types.BlockNumber `scale:"2"`
	// The relay-chain block this will time-out at, if any.
	TimeoutAt types.BlockNumber `scale:"3"`
	// If this core is freed by being timed-out, this is the assignment that is next up on this
	// core. None if there is nothing queued for this core or there is no possibility of timing
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
	ParaID ParaID
	// The collator required to author the block, if any.
	Collator *CollatorID
}

// Index returns the index
func (ScheduledCore) Index() uint {
	return 1
}

// Free Core information about a core which is currently free.
type Free struct {
}

// Index returns the index
func (Free) Index() uint {
	return 2
}

// CoreState represents the state of a particular availability core.
type CoreState scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (va *CoreState) Set(val scale.VaryingDataTypeValue) (err error) {
	// cast to VaryingDataType to use VaryingDataType.Set method
	vdt := scale.VaryingDataType(*va)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	// store original ParentVDT with VaryingDataType that has been set
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

// headData is Parachain head data included in the chain.
type headData []byte

// CandidateCommitments are Commitments made in a `CandidateReceipt`. Many of these are outputs of validation.
type CandidateCommitments struct {
	// Messages destined to be interpreted by the Relay chain itself.
	UpwardMessages []UpwardMessage `scale:"1"`
	// Horizontal messages sent by the parachain.
	HorizontalMessages []OutboundHrmpMessage `scale:"2"`
	// New validation code.
	NewValidationCode ValidationCode `scale:"3"`
	// The head-data produced as a result of execution.
	HeadData headData `scale:"4"`
	// The number of messages processed from the DMQ.
	ProcessedDownwardMessages uint32 `scale:"5"`
	// The mark which specifies the block number up to which all inbound HRMP messages are processed.
	HrmpWatermark uint32 `scale:"6"`
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

// AssignmentID The public key of a keypair used by a validator for determining assignments
// to approve included parachain candidates.
type AssignmentID [32]byte

// IndexedValidator A validator with its index.
type IndexedValidator struct {
	Index     []types.ValidatorIndex `scale:"-"`
	Validator []types.ValidatorID    `scale:"2"`
}

// IndexedValidatorGroup A validator group with its group index.
type IndexedValidatorGroup struct {
	GroupIndex []GroupIndex           `scale:"1"`
	Validators []types.ValidatorIndex `scale:"2"`
}

// AuthorityDiscoveryID An authority discovery key.
type AuthorityDiscoveryID [32]byte

// SessionInfo Information about validator sets of a session.
type SessionInfo struct {
	// All the validators actively participating in parachain consensus.
	// Indices are into the broader validator set.
	ActiveValidatorIndices []types.ValidatorIndex `scale:"1"`
	// A secure random seed for the session, gathered from BABE.
	RandomSeed [32]byte `scale:"2"`
	// The amount of sessions to keep for disputes.
	DisputePeriod SessionIndex `scale:"3"`
	// Validators in canonical ordering.
	Validators []types.ValidatorID `scale:"4"`
	// Validators' authority discovery keys for the session in canonical ordering.
	DiscoveryKeys []AuthorityDiscoveryID `scale:"5"`
	// The assignment keys for validators.
	AssignmentKeys []AssignmentID `scale:"6"`
	// Validators in shuffled ordering - these are the validator groups as produced
	// by the `Scheduler` module for the session and are typically referred to by
	// `GroupIndex`.
	ValidatorGroups [][]types.ValidatorIndex `scale:"7"`
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
	SentAt types.BlockNumber `scale:"1"`
	// The actual downward message to processes.
	Message DownwardMessage `scale:"2"`
}

// InboundHrmpMessage An HRMP message seen from the perspective of a recipient.
type InboundHrmpMessage struct {
	// The block number at which this message was sent.
	// Specifically, it is the block number at which the candidate that sends this message was
	// enacted.
	SentAt types.BlockNumber `scale:"1"`
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
	CoreIndex        uint32           `scale:"3"`
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
