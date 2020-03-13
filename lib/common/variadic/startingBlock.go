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

// StartingBlock represents an optional interface type (int,hash).
type StartingBlock struct {
	value interface{}
}

// NewStartingBlock returns a new optional.Uint32
func NewStartingBlock(startingBlockMsg []byte) (*StartingBlock, error) {
	firstByte := startingBlockMsg[0]
	if firstByte == 0 {
		return &StartingBlock{
			value: common.NewHash(startingBlockMsg[1:]),
		}, nil
	} else if firstByte == 1 {
		num := startingBlockMsg[1:]
		if len(num) < 8 {
			num = common.AppendZeroes(num, 8)
		}

		return &StartingBlock{
			value: binary.LittleEndian.Uint64(num),
		}, nil
	} else {
		return nil, errors.New("invalid start block in BlockRequest")
	}

}

// Value returns the interface value.
func (x *StartingBlock) Value() interface{} {
	return x.value
}
