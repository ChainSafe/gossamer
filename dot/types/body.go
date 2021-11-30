// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Body is the extrinsics(not encoded) inside a state block.
type Body []Extrinsic

// NewBody returns a Body from an Extrinsic array.
func NewBody(e []Extrinsic) *Body {
	body := Body(e)
	return &body
}

// NewBodyFromBytes returns a Body from a SCALE encoded byte array.
func NewBodyFromBytes(b []byte) (*Body, error) {
	exts := [][]byte{}

	if len(b) == 0 {
		return NewBody([]Extrinsic{}), nil
	}

	err := scale.Unmarshal(b, &exts)
	if err != nil {
		return nil, err
	}

	return NewBody(BytesArrayToExtrinsics(exts)), nil
}

// NewBodyFromEncodedBytes returns a new Body from a slice of byte slices that are
// SCALE encoded extrinsics
func NewBodyFromEncodedBytes(exts [][]byte) (*Body, error) {
	// A collection of same-typed values is encoded, prefixed with a compact
	// encoding of the number of items, followed by each item's encoding
	// concatenated in turn.
	// https://substrate.dev/docs/en/knowledgebase/advanced/codec#vectors-lists-series-sets
	enc, err := scale.Marshal(big.NewInt(int64(len(exts))))
	if err != nil {
		return nil, err
	}

	for _, ext := range exts {
		enc = append(enc, ext...)
	}

	return NewBodyFromBytes(enc)
}

// NewBodyFromExtrinsicStrings creates a block body given an array of hex-encoded
// 0x-prefixed strings.
func NewBodyFromExtrinsicStrings(ss []string) (*Body, error) {
	exts := []Extrinsic{}
	for _, s := range ss {
		b, err := common.HexToBytes(s)
		if err == common.ErrNoPrefix {
			b = []byte(s)
		} else if err != nil {
			return nil, err
		}
		exts = append(exts, b)
	}

	return NewBody(exts), nil
}

// DeepCopy creates a new copy of the body.
func (b *Body) DeepCopy() Body {
	newExtrinsics := make([]Extrinsic, 0, len([]Extrinsic(*b)))

	for _, e := range []Extrinsic(*b) {
		temp := make([]byte, len(e))
		copy(temp, e)

		newExtrinsics = append(newExtrinsics, temp)
	}

	return Body(newExtrinsics)
}

// HasExtrinsic returns true if body contains target Extrinsic
func (b *Body) HasExtrinsic(target Extrinsic) (bool, error) {
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
		encext, err := scale.Marshal(currext)
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
func (b *Body) AsEncodedExtrinsics() ([]Extrinsic, error) {
	decodedExts := *b
	ret := make([]Extrinsic, len(decodedExts))
	var err error

	for i, ext := range decodedExts {
		ret[i], err = scale.Marshal(ext)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}
