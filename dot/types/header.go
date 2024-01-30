// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Header is a state block header
type Header struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         uint        `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         Digest      `json:"digest"`
	hash           common.Hash
}

// NewHeader creates a new block header and sets its hash field
func NewHeader(parentHash, stateRoot, extrinsicsRoot common.Hash,
	number uint, digest Digest) (blockHeader *Header) {
	blockHeader = &Header{
		ParentHash:     parentHash,
		Number:         number,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         digest,
	}
	blockHeader.Hash()
	return blockHeader
}

// NewEmptyHeader returns a new header with all zero values
func NewEmptyHeader() *Header {
	return &Header{}
}

// Exists returns a boolean indicating if the header exists
func (bh *Header) Exists() bool {
	exists := bh != nil
	return exists
}

// Empty returns a boolean indicating is the header is empty
func (bh *Header) Empty() bool {
	if !bh.StateRoot.IsEmpty() || !bh.ExtrinsicsRoot.IsEmpty() || !bh.ParentHash.IsEmpty() {
		return false
	}
	return bh.Number == 0 && len(bh.Digest) == 0
}

// DeepCopy returns a deep copy of the header to prevent side effects down the road
func (bh *Header) DeepCopy() (*Header, error) {
	cp := NewEmptyHeader()
	copy(cp.ParentHash[:], bh.ParentHash[:])
	copy(cp.StateRoot[:], bh.StateRoot[:])
	copy(cp.ExtrinsicsRoot[:], bh.ExtrinsicsRoot[:])

	cp.Number = bh.Number

	if len(bh.Digest) > 0 {
		cp.Digest = NewDigest()
		cp.Digest = append(cp.Digest, bh.Digest...)
	}

	return cp, nil
}

// String returns the formatted header as a string
func (bh *Header) String() string {
	return fmt.Sprintf("ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v Hash=%s",
		bh.ParentHash, bh.Number, bh.StateRoot, bh.ExtrinsicsRoot, bh.Digest, bh.Hash())
}

// Hash returns the hash of the block header
// If the internal hash field is nil, it hashes the block and sets the hash field.
// If hashing the header errors, this will panic.
func (bh *Header) Hash() common.Hash {
	if bh.hash == [32]byte{} {
		enc, err := scale.Marshal(*bh)
		if err != nil {
			panic(err)
		}

		hash, err := common.Blake2bHash(enc)
		if err != nil {
			panic(err)
		}

		bh.hash = hash
	}

	return bh.hash
}
