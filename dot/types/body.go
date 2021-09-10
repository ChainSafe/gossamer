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
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/scale"
	scale2 "github.com/ChainSafe/gossamer/pkg/scale"
)

// Body is the encoded extrinsics inside a state block
type Body []byte

// NewBody returns a Body from a byte array
func NewBody(b []byte) *Body {
	body := Body(b)
	return &body
}

// NewBodyFromBytes returns a new Body from a slice of byte slices
func NewBodyFromBytes(exts [][]byte) (*Body, error) {
	enc, err := scale.Encode(exts)
	if err != nil {
		return nil, err
	}

	body := Body(enc)
	return &body, nil
}

// NewBodyFromEncodedBytes returns a new Body from a slice of byte slices that are SCALE encoded extrinsics
func NewBodyFromEncodedBytes(exts [][]byte) (*Body, error) {
	enc, err := scale.Encode(big.NewInt(int64(len(exts))))
	if err != nil {
		return nil, err
	}

	for _, ext := range exts {
		enc = append(enc, ext...)
	}

	body := Body(enc)
	return &body, nil
}

// NewBodyFromExtrinsics creates a block body given an array of extrinsics.
func NewBodyFromExtrinsics(exts []Extrinsic) (*Body, error) {
	enc, err := scale.Encode(ExtrinsicsArrayToBytesArray(exts))
	if err != nil {
		return nil, err
	}

	body := Body(enc)
	return &body, nil
}

// NewBodyFromExtrinsicStrings creates a block body given an array of hex-encoded 0x-prefixed strings.
func NewBodyFromExtrinsicStrings(ss []string) (*Body, error) {
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

	enc, err := scale.Encode(exts)
	if err != nil {
		return nil, err
	}

	body := Body(enc)
	return &body, nil
}

// AsExtrinsics decodes the body into an array of extrinsics
func (b *Body) AsExtrinsics() ([]Extrinsic, error) {
	exts := [][]byte{}

	if len(*b) == 0 {
		return []Extrinsic{}, nil
	}

	dec, err := scale.Decode(*b, exts)
	if err != nil {
		return nil, err
	}

	return BytesArrayToExtrinsics(dec.([][]byte)), nil
}

// AsEncodedExtrinsics decodes the body into an array of SCALE encoded extrinsics
func (b *Body) AsEncodedExtrinsics() ([]Extrinsic, error) {
	exts := [][]byte{}

	if len(*b) == 0 {
		return []Extrinsic{}, nil
	}

	err := scale2.Unmarshal(*b, &exts)
	if err != nil {
		return nil, err
	}

	decodedExts := exts
	ret := make([][]byte, len(decodedExts))

	for i, ext := range decodedExts {
		ret[i], err = scale2.Marshal(ext)
		if err != nil {
			return nil, err
		}
	}

	return BytesArrayToExtrinsics(ret), nil
}

// AsOptional returns the Body as an optional.Body
func (b *Body) AsOptional() *optional.Body {
	ob := optional.CoreBody([]byte(*b))
	return optional.NewBody(true, ob)
}

// HasExtrinsic returns true if body contains target Extrisic
// returns error when fails to encode decoded extrinsic on body
func (b *Body) HasExtrinsic(target Extrinsic) (bool, error) {
	exts, err := b.AsExtrinsics()
	if err != nil {
		return false, err
	}

	// goes through the decreasing order due to the fact that extrinsicsToBody func (lib/babe/build.go)
	// appends the valid transaction extrinsic on the end of the body
	for i := len(exts) - 1; i >= 0; i-- {
		currext := exts[i]

		// if current extrinsic is equal the target then returns true
		if bytes.Equal(target, currext) {
			return true, nil
		}

		//otherwise try to encode and compare
		encext, err := scale.Encode(currext)
		if err != nil {
			return false, fmt.Errorf("fail while scale encode: %w", err)
		}

		if len(encext) >= len(target) && bytes.Equal(target, encext[:len(target)]) {
			return true, nil
		}
	}

	return false, nil
}
