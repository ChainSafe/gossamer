package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// ValidatorID represents a validator ID
type ValidatorID [sr25519.PublicKeyLength]byte

// Validator represents a validator
type Validator struct {
	Key crypto.PublicKey
}

// FromRawSr25519 sets the Validator given ValidatorID. It converts the byte representations of
// the authority public keys into a sr25519.PublicKey.
func (a *Validator) FromRawSr25519(id ValidatorID) error {
	key, err := sr25519.NewPublicKey(id[:])
	if err != nil {
		return err
	}

	a.Key = key
	return nil
}

// ValidatorIDToValidator turns a slice of ValidatorID into a slice of Validator
func ValidatorIDToValidator(ids []ValidatorID) ([]Validator, error) {
	validators := make([]Validator, len(ids))
	for i, r := range ids {
		validators[i] = Validator{}
		err := validators[i].FromRawSr25519(r)
		if err != nil {
			return nil, err
		}
	}

	return validators, nil
}

// ValidatorIndex represents a validator index
type ValidatorIndex uint32

// GroupRotationInfo represents the group rotation info
type GroupRotationInfo struct {
	// SessionStartBlock is the block number at which the session started
	SessionStartBlock uint64 `scale:"1"`
	// GroupRotationFrequency indicates how often groups rotate. 0 means never.
	GroupRotationFrequency uint64 `scale:"2"`
	// Now indicates the current block number.
	Now uint64 `scale:"3"`
}

type ValidatorGroups struct {
	// Validators is an array the validator set Ids
	Validators [][]ValidatorIndex `scale:"1"`
	// GroupRotationInfo is the group rotation info
	GroupRotationInfo GroupRotationInfo `scale:"2"`
}

// ParaID The ID of a para scheduled.
type ParaID uint32

// GroupIndex The unique (during session) index of a validator group.
type GroupIndex uint32

// CollatorID represents a collator ID
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

	// Collator is the collator's relay-chain account ID
	Collator sr25519.PublicKey `scale:"3"`

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

// ScheduledCore Information about a core which is currently occupied.
type ScheduledCore struct {
	// The ID of a para scheduled.
	ParaID ParaID
	// The collator required to author the block, if any.
	Collator *Collator
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
	// TODO: this should be a bitvec
	Availability []byte `scale:"5"`
	// The group assigned to distribute availability pieces of this candidate.
	GroupResponsible GroupIndex `scale:"6"`
	// The hash of the candidate occupying the core.
	CandidateHash common.Hash `scale:"7"`
	// The descriptor of the candidate occupying the core.
	CandidateDescriptor CandidateDescriptor `scale:"8"`
}

// Occupied Core information about a core which is currently occupied.
type Occupied OccupiedCore

// Index returns the index
func (Occupied) Index() uint {
	return 0
}

// Scheduled Core information about a core which is currently scheduled.
type Scheduled ScheduledCore

// Index returns the index
func (Scheduled) Index() uint {
	return 1
}

// Free Core information about a core which is currently free.
type Free scale.VaryingDataType

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

// NewCoreState returns a new CoreState
func NewCoreState() (CoreState, error) {
	vdt, err := scale.NewVaryingDataType(Occupied{}, Scheduled{}, Free{})
	if err != nil {
		return CoreState{}, fmt.Errorf("failed to create varying data type: %w", err)
	}

	return CoreState(vdt), nil
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
	NewValidationCode *ValidationCode `scale:"3"`
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
type AssignmentID sr25519.PublicKey

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
	Validators map[ValidatorIndex]ValidatorID `scale:"4"`
	// Validators' authority discovery keys for the session in canonical ordering.
	DiscoveryKeys []Authority `scale:"5"`
	// The assignment keys for validators.
	AssignmentKeys []AssignmentID `scale:"6"`
	// Validators in shuffled ordering - these are the validator groups as produced
	// by the `Scheduler` module for the session and are typically referred to by
	// `GroupIndex`.
	ValidatorGroups map[GroupIndex][]ValidatorIndex `scale:"7"`
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
