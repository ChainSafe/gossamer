// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
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

// ExtrinsicData is a transaction which embeds the `ctypes.Extrinsic` and has additional functionality.
type ExtrinsicData struct {
	ctypes.Extrinsic
}

// DecodeVersion decodes only the version field of the Extrinsic.
func (e *ExtrinsicData) DecodeVersion(encExt Extrinsic) error {
	decoder := scale.NewDecoder(bytes.NewReader(encExt))
	_, err := decoder.DecodeUintCompact()
	if err != nil {
		return err
	}

	return decoder.Decode(&e.Version)
}
