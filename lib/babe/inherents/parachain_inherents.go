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
func (d *disputeStatement) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*d)
	return vdt.Value()
}

// validDisputeStatementKind is a kind of statements of validity on a candidate.
type validDisputeStatementKind scale.VaryingDataType //skipcq

// Index returns VDT index
func (validDisputeStatementKind) Index() uint { //skipcq
	return 0
}

func (validDisputeStatementKind) String() string { //skipcq
	return "valid dispute statement kind"
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (v *validDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq
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
func (v *validDisputeStatementKind) Value() (scale.VaryingDataTypeValue, error) { //skipcq
	vdt := scale.VaryingDataType(*v)
	return vdt.Value()
}

// ExplicitValidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitValidDisputeStatementKind struct{} //skipcq

// Index returns VDT index
func (explicitValidDisputeStatementKind) Index() uint { //skipcq
	return 0
}

func (explicitValidDisputeStatementKind) String() string { //skipcq:SCC-U1000
	return "explicit valid dispute statement kind"
}

// backingSeconded is a seconded statement on a candidate from the backing phase.
type backingSeconded common.Hash //skipcq

// Index returns VDT index
func (backingSeconded) Index() uint { //skipcq
	return 1
}

func (b backingSeconded) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingSeconded(%s)", common.Hash(b))
}

// backingValid is a valid statement on a candidate from the backing phase.
type backingValid common.Hash //skipcq

// Index returns VDT index
func (backingValid) Index() uint { //skipcq
	return 2
}

func (b backingValid) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("backingValid(%s)", common.Hash(b))
}

// approvalChecking is an approval vote from the approval checking phase.
type approvalChecking struct{} //skipcq

// Index returns VDT index
func (approvalChecking) Index() uint { //skipcq
	return 3
}

func (approvalChecking) String() string { return "approval checking" }

// invalidDisputeStatementKind is a kind of statements of invalidity on a candidate.
type invalidDisputeStatementKind scale.VaryingDataType //skipcq

// Index returns VDT index
func (invalidDisputeStatementKind) Index() uint { //skipcq
	return 1
}

func (invalidDisputeStatementKind) String() string { //skipcq
	return "invalid dispute statement kind"
}

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (in *invalidDisputeStatementKind) Set(val scale.VaryingDataTypeValue) (err error) { //skipcq
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
func (in *invalidDisputeStatementKind) Value() (scale.VaryingDataTypeValue, error) { //skipcq
	vdt := scale.VaryingDataType(*in)
	return vdt.Value()
}

// explicitInvalidDisputeStatementKind is an explicit statement issued as part of a dispute.
type explicitInvalidDisputeStatementKind struct{} //skipcq

// Index returns VDT index
func (explicitInvalidDisputeStatementKind) Index() uint { //skipcq
	return 0
}

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
