// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
