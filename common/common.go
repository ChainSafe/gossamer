// // Copyright 2019 The gossamer Authors
// // This file is part of the gossamer library.
// //
// // The gossamer library is free software: you can redistribute it and/or modify
// // it under the terms of the GNU Lesser General Public License as published by
// // the Free Software Foundation, either version 3 of the License, or
// // (at your option) any later version.
// //
// // The gossamer library is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// // GNU Lesser General Public License for more details.
// //
// // You should have received a copy of the GNU Lesser General Public License
// // along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package common

// Concat concatenates two byte arrays
// used instead of append to prevent modifying the original byte array
func Concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

// Uint16ToBytes converts a uint16 into a 2-byte slice
func Uint16ToBytes(in uint16) (out []byte) {
	out = make([]byte, 2)
	out[0] = byte(in & 0x00ff)
	out[1] = byte(in >> 8 & 0x00ff)
	return out
}