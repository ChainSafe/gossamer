// SPDX-License-Identifier: LGPL-3.0-only
// Copyright 2022 ChainSafe Systems (ON)

package inherents

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// disputeStatement is a statement about a candidate, to be used within the dispute
// resolution process. Statements are either in favour of the candidate's validity
// or against it.
type disputeStatementValues interface {
	validDisputeStatementKind | invalidDisputeStatementKind
}

type disputeStatement struct {
	inner any
}

func setdisputeStatement[Value disputeStatementValues](mvdt *disputeStatement, value Value) {
	mvdt.inner = value
}

func (mvdt *disputeStatement) SetValue(value any) (err error) {
	switch value := value.(type) {
	case validDisputeStatementKind:
		setdisputeStatement(mvdt, value)
		return

	case invalidDisputeStatementKind:
		setdisputeStatement(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt disputeStatement) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case validDisputeStatementKind:
		return 0, mvdt.inner, nil

	case invalidDisputeStatementKind:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt disputeStatement) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt disputeStatement) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(validDisputeStatementKind), nil

	case 1:
		return *new(invalidDisputeStatementKind), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// validDisputeStatementKind is a kind of statements of validity on a candidate.
type validDisputeStatementKind struct {
	inner any
}
type validDisputeStatementKindValues interface {
	explicitValidDisputeStatementKind | backingSeconded | backingValid | approvalChecking
}

func setvalidDisputeStatementKind[Value validDisputeStatementKindValues](mvdt *validDisputeStatementKind, value Value) {
	mvdt.inner = value
}

func (mvdt *validDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case explicitValidDisputeStatementKind:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case backingSeconded:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case backingValid:
		setvalidDisputeStatementKind(mvdt, value)
		return

	case approvalChecking:
		setvalidDisputeStatementKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt validDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case explicitValidDisputeStatementKind:
		return 0, mvdt.inner, nil

	case backingSeconded:
		return 1, mvdt.inner, nil

	case backingValid:
		return 2, mvdt.inner, nil

	case approvalChecking:
		return 3, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt validDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt validDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(explicitValidDisputeStatementKind), nil

	case 1:
		return *new(backingSeconded), nil

	case 2:
		return *new(backingValid), nil

	case 3:
		return *new(approvalChecking), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitValidDisputeStatementKind struct{} //skipcq

func (explicitValidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "explicit valid dispute statement kind"
}

// backingSeconded is a seconded statement on a candidate from the backing phase.
type backingSeconded common.Hash //skipcq

func (b backingSeconded) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingSeconded(%s)", common.Hash(b))
}

// backingValid is a valid statement on a candidate from the backing phase.
type backingValid common.Hash //skipcq

func (b backingValid) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingValid(%s)", common.Hash(b))
}

// approvalChecking is an approval vote from the approval checking phase.
type approvalChecking struct{} //skipcq

func (approvalChecking) String() string { return "approval checking" }

// invalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type invalidDisputeStatementKindValues interface {
	explicitInvalidDisputeStatementKind
}

type invalidDisputeStatementKind struct {
	inner any
}

func setinvalidDisputeStatementKind[Value invalidDisputeStatementKindValues](
	mvdt *invalidDisputeStatementKind, value Value,
) {
	mvdt.inner = value
}

func (mvdt *invalidDisputeStatementKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case explicitInvalidDisputeStatementKind:
		setinvalidDisputeStatementKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt invalidDisputeStatementKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case explicitInvalidDisputeStatementKind:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt invalidDisputeStatementKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}
func (mvdt invalidDisputeStatementKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(explicitInvalidDisputeStatementKind), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func (invalidDisputeStatementKind) String() string { //skipcq
	return "invalid dispute statement kind"
}

// explicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitInvalidDisputeStatementKind struct{} //skipcq

func (explicitInvalidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "explicit invalid dispute statement kind"
}

// newDisputeStatement create a new DisputeStatement varying data type.
func newDisputeStatement() disputeStatement { //skipcq
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

// multiDisputeStatementSet is a set of dispute statements.
type multiDisputeStatementSet []disputeStatementSet

// statement about the candidate.
// Used as translation of `Vec<(DisputeStatement, ValidatorIndex, ValidatorSignature)>` from rust to go
type statement struct {
	ValidatorIndex     parachaintypes.ValidatorIndex
	ValidatorSignature parachaintypes.ValidatorSignature
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
	Bitfields []parachaintypes.UncheckedSignedAvailabilityBitfield `scale:"1"`
	// Backed candidates for inclusion in the block.
	BackedCandidates []parachaintypes.BackedCandidate `scale:"2"`
	// Sets of dispute votes for inclusion,
	Disputes multiDisputeStatementSet `scale:"3"`
	// The parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}
