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
package common

import (
	"encoding/binary"
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

// Uint128FromBigInt returns a new Uint128 from a *big.Int
func Uint128FromBigInt(in *big.Int) *Uint128 {
	bytes := in.Bytes()

	if len(bytes) < 16 {
		bytes = padTo16BytesBE(bytes)
	}

	// *big.Int returns bytes in big endian format
	upper := binary.BigEndian.Uint64(bytes[:8])
	lower := binary.BigEndian.Uint64(bytes[8:])

	return &Uint128{
		Upper: upper,
		Lower: lower,
	}
}

// Uint128FromLEBytes returns a new Uint128 from a little-endian byte slice
// If the slice is greater than 16 bytes long, it only uses the first 16 bytes
func Uint128FromLEBytes(in []byte) *Uint128 {
	if len(in) < 16 {
		in = padTo16BytesLE(in)
	}

	lower := binary.LittleEndian.Uint64(in[:8])
	upper := binary.LittleEndian.Uint64(in[8:])

	return &Uint128{
		Upper: upper,
		Lower: lower,
	}
}

// ToLEBytes returns the Uint128 as a little endian byte slice
func (u *Uint128) ToLEBytes() []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[:8], u.Lower)
	binary.LittleEndian.PutUint64(buf[8:], u.Upper)
	return trimLEBytes(buf)
}

// ToBEBytes returns the Uint128 as a big endian byte slice
func (u *Uint128) ToBEBytes() []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[:8], u.Upper)
	binary.BigEndian.PutUint64(buf[8:], u.Lower)
	return trimBEBytes(buf)
}

// Cmp returns 1 if the receiver is greater than other, 0 if they are equal, and -1 otherwise.
func (u *Uint128) Cmp(other *Uint128) int {
	if u.Upper > other.Upper {
		return 1
	}

	if u.Upper < other.Upper {
		return -1
	}

	if u.Lower > other.Lower {
		return 1
	}

	if u.Lower < other.Lower {
		return -1
	}

	return 0
}

func padTo16BytesLE(in []byte) []byte {
	for len(in) != 16 {
		in = append(in, 0)
	}
	return in
}

func padTo16BytesBE(in []byte) []byte {
	for len(in) != 16 {
		in = append([]byte{0}, in...)
	}
	return in
}

func trimLEBytes(in []byte) []byte {
	for {
		if len(in) == 0 {
			return in
		}

		if in[len(in)-1] == 0 {
			in = in[:len(in)-1]
		} else {
			break
		}
	}
	return in
}

func trimBEBytes(in []byte) []byte {
	for {
		if len(in) == 0 {
			return in
		}

		if in[0] == 0 {
			in = in[1:]
		} else {
			break
		}
	}
	return in
}
