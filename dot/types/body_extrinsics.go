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
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	scale2 "github.com/ChainSafe/gossamer/pkg/scale"
)

// BodyExtrinsics is the extrinsics(not encoded) inside a state block.
type BodyExtrinsics []Extrinsic

// NewBodyExtrinsics returns a BodyExtrinsics from an Extrinsic array.
func NewBodyExtrinsics(e []Extrinsic) *BodyExtrinsics {
	body := BodyExtrinsics(e)
	return &body
}

// NewBodyExtrinsics returns a BodyExtrinsics from a SCALE encoded byte array.
func NewBodyExtrinsicsFromBytes(b []byte) (*BodyExtrinsics, error) {
	exts := [][]byte{}

	if len(b) == 0 {
		return NewBodyExtrinsics([]Extrinsic{}), nil
	}

	err := scale2.Unmarshal(b, &exts)
	if err != nil {
		return nil, err
	}

	return NewBodyExtrinsics(BytesArrayToExtrinsics(exts)), nil
}

// NewBodyExtrinsicsFromEncodedBytes returns a new Body from a slice of byte slices that are
// SCALE encoded extrinsics
func NewBodyExtrinsicsFromEncodedBytes(exts [][]byte) (*BodyExtrinsics, error) {
	// A collection of same-typed values is encoded, prefixed with a compact
	// encoding of the number of items, followed by each item's encoding
	// concatenated in turn.
	// https://substrate.dev/docs/en/knowledgebase/advanced/codec#vectors-lists-series-sets
	enc, err := scale2.Marshal(big.NewInt(int64(len(exts))))
	if err != nil {
		return nil, err
	}

	for _, ext := range exts {
		enc = append(enc, ext...)
	}

	return NewBodyExtrinsicsFromBytes(enc)
}

// NewBodyExtrinsicsFromExtrinsicStrings creates a block body given an array of hex-encoded
// 0x-prefixed strings.
func NewBodyExtrinsicsFromExtrinsicStrings(ss []string) (*BodyExtrinsics, error) {
	exts := [][]byte{}
	for _, s := range ss {
		b, err := common.HexToBytes(s)
		if err == common.ErrNoPrefix {
			b = []byte(s)
		} else if err != nil {
			return nil, err
		}
		exts = append(exts, b)
	}

	return NewBodyExtrinsics(BytesArrayToExtrinsics(exts)), nil
}

func (b *BodyExtrinsics) AsSCALEEncodedBody() (Body, error) {
	encodedBody, err := scale2.Marshal(ExtrinsicsArrayToBytesArray(*b))
	if err != nil {
		return nil, err
	}

	return Body(encodedBody), nil
}

func (b *BodyExtrinsics) DeepCopy() BodyExtrinsics {
	newExtrinsics := make([]Extrinsic, len([]Extrinsic(*b)))
	copy(newExtrinsics, []Extrinsic(*b))

	return BodyExtrinsics(newExtrinsics)
}

// HasExtrinsic returns true if body contains target Extrisic
func (b *BodyExtrinsics) HasExtrinsic(target Extrinsic) (bool, error) {
	exts := *b

	// goes through the decreasing order due to the fact that extrinsicsToBody
	// func (lib/babe/build.go) appends the valid transaction extrinsic on the end of the body
	for i := len(exts) - 1; i >= 0; i-- {
		currext := exts[i]

		// if current extrinsic is equal the target then returns true
		if bytes.Equal(target, currext) {
			return true, nil
		}

		//otherwise try to encode and compare
		encext, err := scale2.Marshal(currext)
		if err != nil {
			return false, fmt.Errorf("fail while scale encode: %w", err)
		}

		if len(encext) >= len(target) && bytes.Equal(target, encext[:len(target)]) {
			return true, nil
		}
	}

	return false, nil
}

// AsEncodedExtrinsics decodes the body into an array of SCALE encoded extrinsics
func (b *BodyExtrinsics) AsEncodedExtrinsics() ([]Extrinsic, error) {
	decodedExts := *b
	ret := make([][]byte, len(decodedExts))
	var err error

	for i, ext := range decodedExts {
		ret[i], err = scale2.Marshal(ext)
		if err != nil {
			return nil, err
		}
	}

	return BytesArrayToExtrinsics(ret), nil
}
