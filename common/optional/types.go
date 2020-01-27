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
	"fmt"

	common "github.com/ChainSafe/gossamer/common"
)

// Uint32 struct
type Uint32 struct {
	exists bool
	value  uint32
}

// NewUint32 new
func NewUint32(exists bool, value uint32) *Uint32 {
	return &Uint32{
		exists: exists,
		value:  value,
	}
}

// Exists bool
func (x *Uint32) Exists() bool {
	return x.exists
}

// Value uint32
func (x *Uint32) Value() uint32 {
	return x.value
}

// String string
func (x *Uint32) String() string {
	return fmt.Sprintf("%d", x.value)
}

// Set bool uint32
func (x *Uint32) Set(exists bool, value uint32) {
	x.exists = exists
	x.value = value
}

// Hash struct
type Hash struct {
	exists bool
	value  common.Hash
}

// NewHash Hash
func NewHash(exists bool, value common.Hash) *Hash {
	return &Hash{
		exists: exists,
		value:  value,
	}
}

// Exists bool
func (x *Hash) Exists() bool {
	return x.exists
}

// Value Hash
func (x *Hash) Value() common.Hash {
	return x.value
}

// String string
func (x *Hash) String() string {
	return fmt.Sprintf("%x", x.value)
}

// Set bool, Hash
func (x *Hash) Set(exists bool, value common.Hash) {
	x.exists = exists
	x.value = value
}
