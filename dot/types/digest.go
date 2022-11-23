// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// NewDigestItem returns a new VaryingDataType to represent a DigestItem
func NewDigestItem() scale.VaryingDataType {
	return scale.MustNewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
}

// NewDigest returns a new Digest as a varying data type slice.
func NewDigest() scale.VaryingDataTypeSlice {
	return scale.NewVaryingDataTypeSlice(NewDigestItem())
}

// DigestToString renders a digest varying data type slice as a string.
func DigestToString(digest scale.VaryingDataTypeSlice) (s string) {
	elements := make([]string, len(digest.Types))
	for i := range digest.Types {
		elements[i] = digest.Types[i].String()
	}
	return strings.Join(elements, ", ")
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

// ChangesTrieRootDigest contains the root of the changes trie at a given block, if the runtime supports it.
type ChangesTrieRootDigest struct {
	Hash common.Hash
}

// Index returns VDT index
func (ChangesTrieRootDigest) Index() uint { return 2 }

// String returns the digest as a string
func (d ChangesTrieRootDigest) String() string {
	return fmt.Sprintf("ChangesTrieRootDigest Hash=%s", d.Hash)
}

// PreRuntimeDigest contains messages from the consensus engine to the runtime.
type PreRuntimeDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

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
type ConsensusDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// Index returns VDT index
func (ConsensusDigest) Index() uint { return 4 }

// String returns the digest as a string
func (d ConsensusDigest) String() string {
	return fmt.Sprintf("ConsensusDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}

// SealDigest contains the seal or signature. This is only used by native code.
type SealDigest struct {
	ConsensusEngineID ConsensusEngineID
	Data              []byte
}

// Index returns VDT index
func (SealDigest) Index() uint { return 5 }

// String returns the digest as a string
func (d SealDigest) String() string {
	return fmt.Sprintf("SealDigest ConsensusEngineID=%s Data=0x%x", d.ConsensusEngineID.ToBytes(), d.Data)
}
