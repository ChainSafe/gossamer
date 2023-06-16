package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Network messages used by the statement distribution subsystem.
type StatementDistributionMessage scale.VaryingDataType

func NewStatementDistributionMessage() StatementDistributionMessage {
	vdt := scale.MustNewVaryingDataType(SignedFullStatement{}, SecondedStatementWithLargePayload{})
	return StatementDistributionMessage(vdt)
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (sdm *StatementDistributionMessage) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*sdm)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*sdm = StatementDistributionMessage(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (sdm *StatementDistributionMessage) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*sdm)
	return vdt.Value()
}

// A signed full statement under a given relay-parent.
type SignedFullStatement struct {
	Hash                         common.Hash                  `scale:"1"`
	UncheckedSignedFullStatement UncheckedSignedFullStatement `scale:"2"`
}

func (s SignedFullStatement) Index() uint {
	return 0
}

// Seconded statement with large payload (e.g. containing a runtime upgrade).
//
// We only gossip the hash in that case, actual payloads can be fetched from sending node
// via request/response.
type SecondedStatementWithLargePayload StatementMetadata

func (l SecondedStatementWithLargePayload) Index() uint {
	return 1
}

// Variant of `SignedFullStatement` where the signature has not yet been verified.
type UncheckedSignedFullStatement struct {
	// The payload is part of the signed data. The rest is the signing context,
	// which is known both at signing and at validation.
	Payload Statement `scale:"1"`

	// The index of the validator signing this statement.
	ValidatorIndex ValidatorIndex `scale:"2"`

	// The signature by the validator of the signed payload.
	Signature ValidatorSignature `scale:"3"`
}

// Index of the validator.
type ValidatorIndex struct {
	Value uint32
}

// Data that makes a statement unique.
type StatementMetadata struct {
	// Relay parent this statement is relevant under.
	RelayParent common.Hash `scale:"1"`

	// Hash of the candidate that got validated.
	CandidateHash CandidateHash `scale:"2"`

	// Validator that attested the validity.
	SignedBy ValidatorIndex `scale:"3"`

	// Signature of seconding validator.
	Signature ValidatorSignature `scale:"4"`
}

// Signature with which parachain validators sign blocks.
type ValidatorSignature Signature
type Signature [64]byte
