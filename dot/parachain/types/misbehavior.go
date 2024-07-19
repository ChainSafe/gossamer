// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

var (
	_ Misbehaviour = (*MultipleCandidates)(nil)
	_ Misbehaviour = (*UnauthorizedStatement)(nil)
	_ Misbehaviour = (*ValidityDoubleVoteIssuedAndValidity)(nil)
	_ Misbehaviour = (*DoubleSignOnSeconded)(nil)
	_ Misbehaviour = (*DoubleSignOnValidity)(nil)
)

// Misbehaviour is intended to represent different kinds of misbehaviour along with supporting proofs.
type Misbehaviour interface {
	IsMisbehaviour()
}

// ValidityDoubleVoteIssuedAndValidity misbehaviour: voting implicitly by issuing and explicit voting for validity.
//
// ValidityDoubleVote misbehaviour: voting more than one way on candidate validity.
// Since there are three possible ways to vote, a double vote is possible in
// three possible combinations (unordered)
type ValidityDoubleVoteIssuedAndValidity struct {
	CommittedCandidateReceiptAndSign CommittedCandidateReceiptAndSign
	CandidateHashAndSign             CandidateHashAndSign
}

func (ValidityDoubleVoteIssuedAndValidity) IsMisbehaviour() {}

// CommittedCandidateReceiptAndSign combines a committed candidate receipt and its associated signature.
type CommittedCandidateReceiptAndSign struct {
	CommittedCandidateReceipt CommittedCandidateReceipt
	Signature                 ValidatorSignature
}

// CandidateHashAndSign combines a candidate hash and its associated signature.
type CandidateHashAndSign struct {
	CandidateHash CandidateHash
	Signature     ValidatorSignature
}

// MultipleCandidates misbehaviour: declaring multiple candidates.
type MultipleCandidates struct {
	First  CommittedCandidateReceiptAndSign
	Second CommittedCandidateReceiptAndSign
}

func (MultipleCandidates) IsMisbehaviour() {}

// UnauthorizedStatement misbehaviour: submitted statement for wrong group.
// A signed statement which was submitted without proper authority.
type UnauthorizedStatement SignedFullStatement

func (UnauthorizedStatement) IsMisbehaviour() {}

// DoubleSignOnSeconded represents a double sign on a candidate.
type DoubleSignOnSeconded struct {
	Candidate CommittedCandidateReceipt
	Sign1     ValidatorSignature
	Sign2     ValidatorSignature
}

func (DoubleSignOnSeconded) IsMisbehaviour() {}

// DoubleSignOnValidity represents a double sign on validity.
type DoubleSignOnValidity struct {
	CandidateHash CandidateHash
	Sign1         ValidatorSignature
	Sign2         ValidatorSignature
}

func (DoubleSignOnValidity) IsMisbehaviour() {}
