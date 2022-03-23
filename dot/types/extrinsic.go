// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// Extrinsic is a generic transaction whose format is verified in the runtime
type Extrinsic []byte

// NewExtrinsic creates a new Extrinsic given a byte slice
func NewExtrinsic(e []byte) Extrinsic {
	return Extrinsic(e)
}

func (e Extrinsic) String() string {
	return common.BytesToHex(e)
}

// Hash returns the blake2b hash of the extrinsic
func (e Extrinsic) Hash() common.Hash {
	hash, err := common.Blake2bHash(e)
	if err != nil {
		panic(err)
	}

	return hash
}

// ExtrinsicsArrayToBytesArray converts an array of extrinsics into an array of byte arrays
func ExtrinsicsArrayToBytesArray(exts []Extrinsic) [][]byte {
	b := make([][]byte, len(exts))
	for i, ext := range exts {
		b[i] = []byte(ext)
	}
	return b
}

// BytesArrayToExtrinsics converts an array of byte arrays into an array of extrinsics
func BytesArrayToExtrinsics(b [][]byte) []Extrinsic {
	exts := make([]Extrinsic, len(b))
	for i, be := range b {
		exts[i] = be
	}
	return exts
}
