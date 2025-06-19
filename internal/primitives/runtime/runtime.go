// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"slices"

	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

// Justification is an abstraction over justification for a block's validity under a consensus algorithm.
//
// Essentially a finality proof. The exact formulation will vary between consensus algorithms. In the case where there
// are multiple valid proofs, inclusion within the block itself would allow swapping justifications to change the
// block's hash (and thus fork the chain). Sending a Justification alongside a block instead bypasses this problem.
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

func (j *Justifications) Append(justification Justification) bool {
	if j.Get(justification.ConsensusEngineID) != nil {
		return false
	}
	*j = append(*j, justification)
	return true
}

func (j Justifications) Get(engineID ConsensusEngineID) *EncodedJustification {
	index := slices.IndexFunc(j, func(j Justification) bool {
		return j.ConsensusEngineID == engineID
	})
	if index >= 0 {
		return &j[index].EncodedJustification
	}
	return nil
}

// IntoJustification returns the encoded justification for the given consensus engine, if it exists.
func (j Justifications) IntoJustification(enginedID ConsensusEngineID) *EncodedJustification {
	for _, justification := range j {
		if justification.ConsensusEngineID == enginedID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

// Complex storage builder stuff.
type BuildStorage interface {
	// Build the storage out of this builder.
	BuildStorage() (storage.Storage, error)
}

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

// OpaqueValue is a simple blob that hold a value in an encoded form without committing to its type.
type OpaqueValue []byte
