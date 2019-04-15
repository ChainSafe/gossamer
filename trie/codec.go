// // Copyright 2019 ChainSafe Systems (ON) Corp.
// // This file is part of gossamer.
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

package trie

// keyEncodeByte swaps the two nibbles of a byte to result in 'LE'
func keyEncodeByte(b byte) byte {
	b1 := (uint(b) & 240) >> 4
	b2 := (uint(b) & 15) << 4

	return byte(b1 | b2)
}

// KeyEncode encodes the key by placing in in little endian nibble format
func KeyEncode(k []byte) []byte {
	result := make([]byte, len(k))
	// Encode each byte
	for i, b := range k {
		result[i] = keyEncodeByte(b)
	}
	return result
}

// keyToHex turns bytes into nibbles
func keyToHex(in []byte) []byte {
	l := len(in) * 2
	res := make([]byte, l)
	for i, b := range in {
		res[2*i] = b / 16
		res[2*i+1] = b % 16
	}

	return res
}