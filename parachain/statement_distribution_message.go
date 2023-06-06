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

type SecondedStatementWithLargePayload StatementMetadata

func (l SecondedStatementWithLargePayload) Index() uint {
	return 1
}

type UncheckedSignedFullStatement struct {
	Payload        Statement          `scale:"1"`
	ValidatorIndex ValidatorIndex     `scale:"2"`
	Signature      ValidatorSignature `scale:"3"`
	RealPayload    CompactStatement   `scale:"4"` // changes needed
}

type PhantomData struct{}

type ValidatorIndex struct {
	Value uint32
}

type StatementMetadata struct {
	RelayParent   common.Hash        `scale:"1"`
	CandidateHash CandidateHash      `scale:"2"`
	SignedBy      ValidatorIndex     `scale:"3"`
	Signature     ValidatorSignature `scale:"4"`
}

type ValidatorSignature Signature
type Signature [64]byte
