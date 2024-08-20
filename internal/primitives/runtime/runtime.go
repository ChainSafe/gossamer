// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

// Justification is an abstraction over justification for a block's validity under a consensus algorithm.
//
// Essentially a finality proof. The exact formulation will vary between consensus algorithms. In the case where there
// are multiple valid proofs, inclusion within the block itself would allow swapping justifications to change the
// block's hash (and thus fork the chain). Sending a `Justification` alongside a block instead bypasses this problem.
//
// Each justification is provided as an encoded blob, and is tagged with an ID to identify the consensus engine that
// generated the proof (we might have multiple justifications from different engines for the same block).
type Justification struct {
	ConsensusEngineID
	EncodedJustification
}

// EncodedJustification is the encoded justification specific to a consensus engine.
type EncodedJustification []byte

// Justifications is a collection of justifications for a given block, multiple justifications may be provided by
// different consensus engines for the same block.
type Justifications []Justification

// EncodedJustification returns a copy of the encoded justification for the given consensus engine, if it exists
func (j Justifications) EncodedJustification(engineID ConsensusEngineID) *EncodedJustification {
	for _, justification := range j {
		if justification.ConsensusEngineID == engineID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

// Consensus engine unique ID.
type ConsensusEngineID [4]byte
