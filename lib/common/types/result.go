// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Result represents a Result type.
type Result struct {
	isErr byte // If non-error result then isErr stores byte(0), otherwise byte(1)
	data  []byte
}

// NewResult returns a new Result type
func NewResult(isErr byte, data []byte) *Result {
	return &Result{
		isErr: isErr,
		data:  data,
	}
}

// Encode returns the SCALE encoded Result
func (r *Result) Encode() ([]byte, error) {
	if r == nil || r.isErr == 1 {
		return []byte{1}, nil
	}

	return append([]byte{0}, r.data...), nil
}

// Decode return a Result from scale encoded data
func (r *Result) Decode(reader io.Reader) (*Result, error) {
	exists, err := common.ReadByte(reader)
	if err != nil {
		return nil, err
	}

	if exists > 1 {
		return nil, ErrInvalidResult
	}

	r.isErr = exists

	if r.isErr == 1 {
		return r, nil
	}

	r.data = []byte{}

	for {
		b, err := common.ReadByte(reader)
		if err != nil {
			break
		}

		r.data = append(r.data, b)
	}

	return r, nil
}

// Value returns the []byte data. It returns nil if isErr is true.
func (r *Result) Value() []byte {
	return r.data
}
