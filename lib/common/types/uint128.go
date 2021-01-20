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
	"encoding/binary"
	"math/big"
)

// Uint128 represents an unsigned 128 bit integer
type Uint128 struct {
	upper uint64
	lower uint64
}

// MaxUint128 is the maximum uint128 value
var MaxUint128 = &Uint128{
	upper: ^uint64(0),
	lower: ^uint64(0),
}

// Uint128FromBigInt returns a new Uint128 from a *big.Int
func Uint128FromBigInt(in *big.Int) *Uint128 {
	bytes := in.Bytes()

	if len(bytes) < 16 {
		bytes = padTo16Bytes(bytes)
	}

	// *big.Int returns bytes in big endian format
	upper := binary.BigEndian.Uint64(bytes[:8])
	lower := binary.BigEndian.Uint64(bytes[8:])

	return &Uint128{
		upper: upper,
		lower: lower,
	}
}

// Uint128FromLEBytes returns a new Uint128 from a little-endian byte slice
// If the slice is greater than 16 bytes long, it only uses the first 16 bytes
func Uint128FromLEBytes(in []byte) *Uint128 {
	if len(in) < 16 {
		in = padTo16Bytes(in)
	}

	lower := binary.LittleEndian.Uint64(in[:8])
	upper := binary.LittleEndian.Uint64(in[8:])

	return &Uint128{
		upper: upper,
		lower: lower,
	}
}

// String returns the Uint128 as a decimal string
func (u *Uint128) String() string {
	upper := big.NewInt(int64(u.upper))
	upper = new(big.Int).Lsh(upper, 64)
	lower := big.NewInt(int64(u.lower))
	return new(big.Int).Or(upper, lower).String()
}

// ToLEBytes returns the Uint128 as a little endian byte slice
func (u *Uint128) ToLEBytes() []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[:8], u.lower)
	binary.LittleEndian.PutUint64(buf[8:], u.upper)
	return buf
}

// Cmp returns 1 if the receiver is greater than other, 0 if they are equal, and -1 otherwise.
func (u *Uint128) Cmp(other *Uint128) int {
	if u.upper > other.upper {
		return 1
	}

	if u.upper < other.upper {
		return -1
	}

	if u.lower > other.lower {
		return 1
	}

	if u.lower < other.lower {
		return -1
	}

	return 0
}

func padTo16Bytes(in []byte) []byte {
	for len(in) != 16 {
		in = append(in, 0)
	}
	return in
}
