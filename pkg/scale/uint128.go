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

package scale

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

// Uint128 represents an unsigned 128 bit integer
type Uint128 struct {
	Upper uint64
	Lower uint64
}

// MaxUint128 is the maximum uint128 value
var MaxUint128 = &Uint128{
	Upper: ^uint64(0),
	Lower: ^uint64(0),
}

// MustNewUint128 will panic if NewUint128 returns an error
func MustNewUint128(in interface{}, order ...binary.ByteOrder) (u *Uint128) {
	u, err := NewUint128(in, order...)
	if err != nil {
		panic(err)
	}
	return
}

func padBytes(b []byte, order binary.ByteOrder) []byte {
	for len(b) != 16 {
		switch order {
		case binary.BigEndian:
			b = append([]byte{0}, b...)
		case binary.LittleEndian:
			b = append(b, 0)
		}
	}
	return b
}

// NewUint128 is constructor for Uint128 that accepts an option binary.ByteOrder
// option is only used when inputted interface{} is of type []byte
// by default binary.LittleEndian is used for []byte since this is SCALE
func NewUint128(in interface{}, order ...binary.ByteOrder) (u *Uint128, err error) {
	switch in := in.(type) {
	case *big.Int:
		bytes := in.Bytes()
		if len(bytes) < 16 {
			bytes = padBytes(bytes, binary.BigEndian)
		}
		u = &Uint128{
			Upper: binary.BigEndian.Uint64(bytes[:8]),
			Lower: binary.BigEndian.Uint64(bytes[8:]),
		}
	case []byte:
		var o binary.ByteOrder = binary.LittleEndian
		if len(order) > 0 {
			o = order[0]
		}
		if len(in) < 16 {
			in = padBytes(in, o)
		}
		u = &Uint128{
			Upper: o.Uint64(in[8:]),
			Lower: o.Uint64(in[:8]),
		}
	default:
		err = fmt.Errorf("unsupported type: %T", in)
	}
	return
}

// Bytes returns the Uint128 in little endian format by default.  A variadic parameter
// order can be used to specify the binary.ByteOrder used
func (u *Uint128) Bytes(order ...binary.ByteOrder) (b []byte) {
	var o binary.ByteOrder = binary.LittleEndian
	if len(order) > 0 {
		o = order[0]
	}
	b = make([]byte, 16)
	switch o {
	case binary.LittleEndian:
		o.PutUint64(b[:8], u.Lower)
		o.PutUint64(b[8:], u.Upper)
		b = u.trimBytes(b, o)
	case binary.BigEndian:
		o.PutUint64(b[:8], u.Upper)
		o.PutUint64(b[8:], u.Lower)
		b = u.trimBytes(b, o)
	}
	return
}

// Compare returns 1 if the receiver is greater than other, 0 if they are equal, and -1 otherwise.
func (u *Uint128) Compare(other *Uint128) int {
	switch {
	case u.Upper > other.Upper:
		return 1
	case u.Upper < other.Upper:
		return -1
	case u.Upper == other.Upper:
		switch {
		case u.Lower > other.Lower:
			return 1
		case u.Lower < other.Lower:
			return -1
		}
	}
	return 0
}

func (u *Uint128) trimBytes(b []byte, order binary.ByteOrder) []byte {
	switch order {
	case binary.LittleEndian:
		for {
			if len(b) == 0 {
				return b
			}
			if b[len(b)-1] == 0 {
				b = b[:len(b)-1]
			} else {
				break
			}
		}
	case binary.BigEndian:
		for {
			if len(b) == 0 {
				return b
			}
			if b[0] == 0 {
				b = b[1:]
			} else {
				break
			}
		}
	}
	return b
}
