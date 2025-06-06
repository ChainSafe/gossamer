package parachaintypes

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type CompactStatementValues interface {
	Valid | SecondedCandidateHash
}

// EncodableCompactStatement is a helper struct that is used to encode/decode CompactStatement.
type EncodableCompactStatement struct {
	CompactStmt CompactStatementInner
}

func (d *EncodableCompactStatement) ToCompact() (CompactStatement, error) {
	value, err := d.CompactStmt.Value()
	if err != nil {
		return nil, err
	}

	switch inner := value.(type) {
	case Valid:
		return NewCompactValid(CandidateHash(inner)), nil
	case SecondedCandidateHash:
		return NewCompactSeconded(CandidateHash(inner)), nil
	default:
		return nil, fmt.Errorf("unsupported type: %T{%v}", inner, inner)
	}
}

func (d *EncodableCompactStatement) UnmarshalSCALE(reader io.Reader) error {
	decoder := scale.NewDecoder(reader)

	var magicBytes [4]byte
	err := decoder.Decode(&magicBytes)
	if err != nil {
		return err
	}

	if !bytes.Equal(magicBytes[:], backingStatementMagic[:]) {
		return fmt.Errorf("invalid magic bytes")
	}

	var inner CompactStatementInner
	err = decoder.Decode(&inner)
	if err != nil {
		return fmt.Errorf("decoding compactStatementInner: %w", err)
	}

	d.CompactStmt = inner
	return nil
}

func (d EncodableCompactStatement) MarshalSCALE() ([]byte, error) {
	buffer := bytes.NewBuffer(backingStatementMagic[:])
	encoder := scale.NewEncoder(buffer)

	err := encoder.Encode(d.CompactStmt)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

type CompactStatementInner struct {
	inner any
}

func setCompactStatement[Value CompactStatementValues](mvdt *CompactStatementInner, value Value) {
	mvdt.inner = value
}

func (mvdt *CompactStatementInner) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Valid:
		setCompactStatement(mvdt, value)
		return
	case SecondedCandidateHash:
		setCompactStatement(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type: %T{%v}", value, value)
	}
}

func (mvdt CompactStatementInner) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Valid:
		return 2, mvdt.inner, nil
	case SecondedCandidateHash:
		return 1, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CompactStatementInner) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CompactStatementInner) ValueAt(index uint) (value any, err error) {
	switch index {
	case 2:
		return Valid{}, nil
	case 1:
		return SecondedCandidateHash{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// CompactStatement is a compact representation of a statement that can be made about parachain candidates.
// this is the actual value that is signed.
type CompactStatement interface {
	CandidateHash() CandidateHash
	SetCandidateHash(hash CandidateHash)
	ToEncodable() *EncodableCompactStatement
}

var (
	_ CompactStatement = (*CompactValid)(nil)
	_ CompactStatement = (*CompactSeconded)(nil)
)

type CompactValid CandidateHash

func NewCompactValid(hash CandidateHash) *CompactValid {
	cv := new(CompactValid)
	*cv = CompactValid(hash)
	return cv
}

func (v *CompactValid) SetCandidateHash(hash CandidateHash) {
	*v = CompactValid(hash)
}

func (v *CompactValid) CandidateHash() CandidateHash {
	return CandidateHash(*v)
}

func (v *CompactValid) ToEncodable() *EncodableCompactStatement {
	cinner := CompactStatementInner{}
	err := cinner.SetValue(Valid(*v))
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err.Error()))
	}

	return &EncodableCompactStatement{CompactStmt: cinner}
}

type CompactSeconded CandidateHash

func NewCompactSeconded(hash CandidateHash) *CompactSeconded {
	cs := new(CompactSeconded)
	*cs = CompactSeconded(hash)
	return cs
}

func (sch *CompactSeconded) SetCandidateHash(hash CandidateHash) {
	*sch = CompactSeconded(hash)
}

func (sch *CompactSeconded) CandidateHash() CandidateHash {
	return CandidateHash(*sch)
}

func (sch *CompactSeconded) ToEncodable() *EncodableCompactStatement {
	cinner := CompactStatementInner{}
	err := cinner.SetValue(SecondedCandidateHash(*sch))
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err.Error()))
	}

	return &EncodableCompactStatement{CompactStmt: cinner}
}
