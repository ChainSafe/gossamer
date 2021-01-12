// Copyright 2021 ChainSafe Systems (ON) Corp.
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
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// Result represents a Result type.
type Result struct {
	isErr byte // If data exists then isErr stores byte(0), otherwise byte(1)
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

	value, err := scale.Encode(r.data)
	if err != nil {
		return nil, err
	}

	return append([]byte{0}, value...), nil
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

	if r.isErr == 0 {
		sd := scale.Decoder{Reader: reader}
		value, err := sd.DecodeByteArray()
		if err != nil {
			return nil, err
		}
		r.data = value
	}

	return r, nil
}

// Value returns the []byte data. It returns nil if it is Result.None.
func (r *Result) Value() []byte {
	return r.data
}
