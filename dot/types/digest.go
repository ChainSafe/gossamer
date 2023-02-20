// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// DigestItem is a varying date type that holds type identifier and a scaled encoded message payload.
type DigestItem struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// NewDigestItem returns a new VaryingDataType to represent a DigestItem
func NewDigestItem() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
}

// NewDigest returns a new Digest as a varying data type slice.
func NewDigest() scale.VaryingDataTypeSlice {
	return scale.NewVaryingDataTypeSlice(NewDigestItem())
}

// ConsensusEngineID is a 4-character identifier of the consensus engine that produced the digest.
type ConsensusEngineID [4]byte

// ToBytes turns ConsensusEngineID to a byte array
func (h ConsensusEngineID) ToBytes() []byte {
	b := [4]byte(h)
	return b[:]
}

func (h ConsensusEngineID) String() string {
	return fmt.Sprintf("0x%x", h.ToBytes())
}

// BabeEngineID is the hard-coded babe ID
var BabeEngineID = ConsensusEngineID{'B', 'A', 'B', 'E'}

// GrandpaEngineID is the hard-coded grandpa ID
var GrandpaEngineID = ConsensusEngineID{'F', 'R', 'N', 'K'}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest DigestItem

// Index returns VDT index
func (PreRuntimeDigest) Index() uint { return 6 }

// NewBABEPreRuntimeDigest returns a PreRuntimeDigest with the BABE consensus ID
func NewBABEPreRuntimeDigest(data []byte) *PreRuntimeDigest {
	return &PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              data,
	}
}

// String returns the digest as a string
func (d PreRuntimeDigest) String() string {
	return fmt.Sprintf("PreRuntimeDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// ConsensusDigest contains messages from the runtime to the consensus engine.
type ConsensusDigest DigestItem

// Index returns VDT index
func (ConsensusDigest) Index() uint { return 4 }

// String returns the digest as a string
func (d ConsensusDigest) String() string {
	return fmt.Sprintf("ConsensusDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest DigestItem

// Index returns VDT index
func (SealDigest) Index() uint { return 5 }

// String returns the digest as a string
func (d SealDigest) String() string {
	return fmt.Sprintf("SealDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}
