package parachaintypes

var (
	_ Misbehaviour       = (*MultipleCandidates)(nil)
	_ Misbehaviour       = (*UnauthorizedStatement)(nil)
	_ Misbehaviour       = (*IssuedAndValidity)(nil)
	_ Misbehaviour       = (*OnSeconded)(nil)
	_ Misbehaviour       = (*OnValidity)(nil)
	_ DoubleSign         = (*OnSeconded)(nil)
	_ DoubleSign         = (*OnValidity)(nil)
	_ ValidityDoubleVote = (*IssuedAndValidity)(nil)
)

// Misbehaviour is intended to represent different kinds of misbehaviour along with supporting proofs.
type Misbehaviour interface {
	IsMisbehaviour()
}

// ValidityDoubleVote misbehaviour: voting more than one way on candidate validity.
// Since there are three possible ways to vote, a double vote is possible in
// three possible combinations (unordered)
type ValidityDoubleVote interface {
	Misbehaviour
	IsValidityDoubleVote()
}

// IssuedAndValidity represents an implicit vote by issuing and explicit voting for validity.
type IssuedAndValidity struct {
	CommittedCandidateReceiptAndSign CommittedCandidateReceiptAndSign
	CandidateHashAndSign             struct {
		CandidateHash CandidateHash
		Signature     ValidatorSignature
	}
}

func (IssuedAndValidity) IsMisbehaviour()       {}
func (IssuedAndValidity) IsValidityDoubleVote() {}

// CommittedCandidateReceiptAndSign combines a committed candidate receipt and its associated signature.
type CommittedCandidateReceiptAndSign struct {
	CommittedCandidateReceipt CommittedCandidateReceipt
	Signature                 ValidatorSignature
}

// MultipleCandidates misbehaviour: declaring multiple candidates.
type MultipleCandidates struct {
	First  CommittedCandidateReceiptAndSign
	Second CommittedCandidateReceiptAndSign
}

func (MultipleCandidates) IsMisbehaviour() {}

// SignedStatement represents signed statements about candidates.
type SignedStatement struct {
	Statement StatementVDT       `scale:"1"`
	Signature ValidatorSignature `scale:"2"`
	Sender    ValidatorIndex     `scale:"3"`
}

// UnauthorizedStatement misbehaviour: submitted statement for wrong group.
type UnauthorizedStatement struct {
	// A signed statement which was submitted without proper authority.
	Statement SignedStatement
}

func (UnauthorizedStatement) IsMisbehaviour() {}

// DoubleSign misbehaviour: multiple signatures on same statement.
type DoubleSign interface {
	Misbehaviour
	IsDoubleSign()
}

// OnSeconded represents a double sign on a candidate.
type OnSeconded struct {
	Candidate CommittedCandidateReceipt
	Sign1     ValidatorSignature
	Sign2     ValidatorSignature
}

func (OnSeconded) IsMisbehaviour() {}
func (OnSeconded) IsDoubleSign()   {}

// OnValidity represents a double sign on validity.
type OnValidity struct {
	CandidateHash CandidateHash
	Sign1         ValidatorSignature
	Sign2         ValidatorSignature
}

func (OnValidity) IsMisbehaviour() {}
func (OnValidity) IsDoubleSign()   {}
