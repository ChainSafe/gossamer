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

package variadic

import (
	"encoding/binary"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Uint64OrHash represents an optional interface type (int,hash).
type Uint64OrHash struct {
	value interface{}
}

// NewUint64OrHash returns a new optional.Uint32
func NewUint64OrHash(data []byte) (*Uint64OrHash, error) {
	firstByte := data[0]
	if firstByte == 0 {
		return &Uint64OrHash{
			value: common.NewHash(data[1:]),
		}, nil
	} else if firstByte == 1 {
		num := data[1:]
		if len(num) < 8 {
			num = common.AppendZeroes(num, 8)
		}

		return &Uint64OrHash{
			value: binary.LittleEndian.Uint64(num),
		}, nil
	} else {
		return nil, errors.New("invalid start block in BlockRequest")
	}

}

// Value returns the interface value.
func (x *Uint64OrHash) Value() interface{} {
	return x.value
}
