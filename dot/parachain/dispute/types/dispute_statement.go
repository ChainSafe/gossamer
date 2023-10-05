package types

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/ChainSafe/gossamer/dot/parachain"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// SecondedCompactStatement is the proposal of a parachain candidate.
type SecondedCompactStatement struct {
	CandidateHash common.Hash
}

// Index returns the index of the type SecondedCompactStatement.
func (SecondedCompactStatement) Index() uint {
	return 0
}

// ValidCompactStatement represents a valid candidate.
type ValidCompactStatement struct {
	CandidateHash common.Hash
}

// Index returns the index of the type ValidCompactStatement.
func (ValidCompactStatement) Index() uint {
	return 1
}

// CompactStatement is the statement that can be made about parachain candidates
// These are the actual values that are signed.
type CompactStatement scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (cs *CompactStatement) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*cs)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*cs = CompactStatement(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (cs *CompactStatement) Value() (val scale.VaryingDataTypeValue, err error) {
	vdt := scale.VaryingDataType(*cs)
	val, err = vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from varying data type: %w", err)
	}
	return val, nil
}

type SigningContext struct {
	SessionIndex  parachainTypes.SessionIndex
	CandidateHash common.Hash
}

func (cs *CompactStatement) SigningPayload(signingContext SigningContext) ([]byte, error) {
	// scale encode the compact statement
	encodedStatement, err := scale.Marshal(*cs)
	if err != nil {
		return nil, fmt.Errorf("encode compact statement: %w", err)
	}

	// scale encode the signing context
	encodedSigningContext, err := scale.Marshal(signingContext)
	if err != nil {
		return nil, fmt.Errorf("encode signing context: %w", err)
	}

	// concatenate the encoded statement and encoded signing context
	encoded := append(encodedStatement, encodedSigningContext...)
	return encoded, nil
}

// NewCompactStatement creates a new CompactStatement.
func NewCompactStatement() (CompactStatement, error) {
	vdt, err := scale.NewVaryingDataType(ValidCompactStatement{}, SecondedCompactStatement{})
	if err != nil {
		return CompactStatement{}, fmt.Errorf("failed to create varying data type: %w", err)
	}

	return CompactStatement(vdt), nil
}

type CompactStatementKind uint

const (
	SecondedCompactStatementKind CompactStatementKind = iota
	ValidCompactStatementKind
)

func NewCustomCompactStatement(kind CompactStatementKind, candidateHash common.Hash) (CompactStatement, error) {
	vdt, err := NewCompactStatement()
	if err != nil {
		return CompactStatement{}, fmt.Errorf("create new compact statement: %w", err)
	}

	switch kind {
	case SecondedCompactStatementKind:
		err = vdt.Set(SecondedCompactStatement{
			CandidateHash: candidateHash,
		})
	case ValidCompactStatementKind:
		err = vdt.Set(ValidCompactStatement{
			CandidateHash: candidateHash,
		})
	default:
		return CompactStatement{}, fmt.Errorf("invalid compact statement kind")
	}

	if err != nil {
		return CompactStatement{}, fmt.Errorf("set compact statement: %w", err)
	}

	return vdt, nil
}

func NewCompactStatementFromAttestation(
	attestation inherents.ValidityAttestation,
	candidateHash common.Hash,
) (CompactStatement, error) {
	compactStatementVDT, err := NewCompactStatement()
	if err != nil {
		return CompactStatement{}, fmt.Errorf("create new compact statement: %w", err)
	}

	attestationVDT := scale.VaryingDataType(attestation)
	val, err := attestationVDT.Value()
	if err != nil {
		return CompactStatement{}, fmt.Errorf("get attestation value: %w", err)
	}

	switch val.(type) {
	case inherents.Implicit:
		err = compactStatementVDT.Set(ValidCompactStatement{
			CandidateHash: candidateHash,
		})
	case inherents.Explicit:
		err = compactStatementVDT.Set(SecondedCompactStatement{
			CandidateHash: candidateHash,
		})
	default:
		return CompactStatement{}, fmt.Errorf("invalid compact statement kind")
	}
	return compactStatementVDT, nil
}

// ExplicitDisputeStatement An explicit statement on a candidate issued as part of a dispute.
type ExplicitDisputeStatement struct {
	Valid         bool
	CandidateHash common.Hash
	Session       parachainTypes.SessionIndex
}

func (eds ExplicitDisputeStatement) SigningPayload() ([]byte, error) {
	const magic = "DISP"
	var magicBytes [4]byte
	copy(magicBytes[:], magic)

	encoded, err := scale.Marshal(eds)
	if err != nil {
		return nil, fmt.Errorf("marshal ExplicitDisputeStatement")
	}

	// how to return
	payload := append(magicBytes[:], encoded...)
	return payload, nil
}

// ApprovalVote A vote of approval on a candidate.
type ApprovalVote struct {
	candidateHash common.Hash
}

func (a *ApprovalVote) SigningPayload() ([]byte, error) {
	const magic = "APPR"
	var magicBytes [4]byte
	copy(magicBytes[:], magic)

	encoded, err := scale.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("marshal ExplicitDisputeStatement")
	}

	// how to return
	payload := append(magicBytes[:], encoded...)
	return payload, nil
}

// SignedDisputeStatement A checked dispute statement from an associated validator.
type SignedDisputeStatement struct {
	DisputeStatement   inherents.DisputeStatement
	CandidateHash      common.Hash
	ValidatorPublic    parachainTypes.ValidatorID
	ValidatorSignature parachain.ValidatorSignature
	SessionIndex       parachainTypes.SessionIndex
}

func NewSignedDisputeStatement(
	keypair keystore.KeyPair,
	valid bool,
	candidateHash common.Hash,
	sessionIndex parachainTypes.SessionIndex,
) (SignedDisputeStatement, error) {
	disputeStatement := inherents.NewDisputeStatement()
	if valid {
		if err := disputeStatement.Set(inherents.ExplicitValidDisputeStatementKind{}); err != nil {
			return SignedDisputeStatement{}, fmt.Errorf("set dispute statement: %w", err)
		}
	} else {
		if err := disputeStatement.Set(inherents.ExplicitInvalidDisputeStatementKind{}); err != nil {
			return SignedDisputeStatement{}, fmt.Errorf("set dispute statement: %w", err)
		}
	}

	payload, err := getDisputeStatementSigningPayload(disputeStatement, candidateHash, sessionIndex)
	if err != nil {
		return SignedDisputeStatement{}, fmt.Errorf("get dispute statement signing payload: %w", err)
	}

	signature, err := keypair.Sign(payload)
	if err != nil {
		return SignedDisputeStatement{}, fmt.Errorf("sign payload: %w", err)
	}

	return SignedDisputeStatement{
		DisputeStatement:   disputeStatement,
		CandidateHash:      candidateHash,
		ValidatorPublic:    parachainTypes.ValidatorID(keypair.Public().Encode()),
		ValidatorSignature: parachain.ValidatorSignature(signature),
		SessionIndex:       sessionIndex,
	}, nil
}

func NewCheckedSignedDisputeStatement(disputeStatement inherents.DisputeStatement,
	candidateHash common.Hash,
	sessionIndex parachainTypes.SessionIndex,
	validatorPublic parachainTypes.ValidatorID,
	validatorSignature parachain.ValidatorSignature,
) (*SignedDisputeStatement, error) {
	if err := VerifyDisputeStatement(disputeStatement, candidateHash, sessionIndex, validatorPublic, validatorSignature); err != nil {
		return nil, fmt.Errorf("verify dispute statement: %w", err)
	}

	return &SignedDisputeStatement{
		DisputeStatement:   disputeStatement,
		CandidateHash:      candidateHash,
		ValidatorPublic:    validatorPublic,
		ValidatorSignature: validatorSignature,
		SessionIndex:       sessionIndex,
	}, nil
}

func VerifyDisputeStatement(
	disputeStatement inherents.DisputeStatement,
	candidateHash common.Hash,
	sessionIndex parachainTypes.SessionIndex,
	validatorPublic parachainTypes.ValidatorID,
	validatorSignature parachain.ValidatorSignature,
) error {
	payload, err := getDisputeStatementSigningPayload(disputeStatement, candidateHash, sessionIndex)
	if err != nil {
		return fmt.Errorf("get dispute statement signing payload: %w", err)
	}

	if err := validatorSignature.Verify(payload, validatorPublic); err != nil {
		return fmt.Errorf("verify validator signature: %w", err)
	}

	return nil
}

func getDisputeStatementSigningPayload(
	disputeStatement inherents.DisputeStatement,
	candidateHash common.Hash,
	session parachainTypes.SessionIndex,
) ([]byte, error) {
	statement, err := disputeStatement.Value()
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute statement value: %w", err)
	}

	var payload []byte
	switch statement.(type) {
	case inherents.ExplicitValidDisputeStatementKind:
		data := ExplicitDisputeStatement{
			Valid:         true,
			CandidateHash: candidateHash,
			Session:       session,
		}
		payload, err = data.SigningPayload()
		if err != nil {
			return nil, fmt.Errorf("get signing payload: %w", err)
		}
	case inherents.BackingSeconded:
		data, err := NewCustomCompactStatement(SecondedCompactStatementKind, candidateHash)
		if err != nil {
			return nil, fmt.Errorf("new custom compact statement: %w", err)
		}

		signingContext := SigningContext{
			SessionIndex:  session,
			CandidateHash: candidateHash,
		}
		payload, err = data.SigningPayload(signingContext)
		if err != nil {
			return nil, fmt.Errorf("get signing payload: %w", err)
		}
	case inherents.BackingValid:
		data, err := NewCustomCompactStatement(ValidCompactStatementKind, candidateHash)
		if err != nil {
			return nil, fmt.Errorf("new custom compact statement: %w", err)
		}

		signingContext := SigningContext{
			SessionIndex:  session,
			CandidateHash: candidateHash,
		}
		payload, err = data.SigningPayload(signingContext)
		if err != nil {
			return nil, fmt.Errorf("get signing payload: %w", err)
		}
	case inherents.ApprovalChecking:
		data := ApprovalVote{
			candidateHash: candidateHash,
		}
		payload, err = data.SigningPayload()
		if err != nil {
			return nil, fmt.Errorf("get signing payload: %w", err)
		}
	case inherents.InvalidDisputeStatementKind:
		data := ExplicitDisputeStatement{
			Valid:         false,
			CandidateHash: candidateHash,
			Session:       session,
		}
		payload, err = data.SigningPayload()
		if err != nil {
			return nil, fmt.Errorf("get signing payload: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid dispute statement kind")
	}

	return payload, nil
}

// Statement is the statement that can be made about parachain candidates.
type Statement struct {
	SignedDisputeStatement SignedDisputeStatement
	ValidatorIndex         parachainTypes.ValidatorIndex
}
