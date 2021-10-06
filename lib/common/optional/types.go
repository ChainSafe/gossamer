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

package optional

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ChainSafe/gossamer/lib/common"
)

const none = "None"

// FixedSizeBytes represents an optional FixedSizeBytes type. It does not length-encode the value when encoding.
type FixedSizeBytes struct {
	exists bool
	value  []byte
}

// NewFixedSizeBytes returns a new optional.FixedSizeBytes
func NewFixedSizeBytes(exists bool, value []byte) *FixedSizeBytes {
	return &FixedSizeBytes{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *FixedSizeBytes) Exists() bool {
	return x.exists
}

// Value returns the []byte value. It returns nil if it is None.
func (x *FixedSizeBytes) Value() []byte {
	return x.value
}

// String returns the value as a string.
func (x *FixedSizeBytes) String() string {
	if !x.exists {
		return none
	}
	return fmt.Sprintf("%x", x.value)
}

// Set sets the exists and value fields.
func (x *FixedSizeBytes) Set(exists bool, value []byte) {
	x.exists = exists
	x.value = value
}

// Encode returns the SCALE encoded optional
func (x *FixedSizeBytes) Encode() ([]byte, error) {
	if x == nil || !x.exists {
		return []byte{0}, nil
	}

	return append([]byte{1}, x.value...), nil
}

// Decode return an optional FixedSizeBytes from scale encoded data
func (x *FixedSizeBytes) Decode(r io.Reader) (*FixedSizeBytes, error) {
	exists, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	if exists > 1 {
		return nil, errors.New("decoding failed, invalid optional")
	}

	x.exists = exists != 0

	if x.exists {
		value, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		x.value = value
	}

	return x, nil
}
