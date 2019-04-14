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

package trie

type node interface {
	Encode() ([]byte, error)
}

type (
	branch struct {
		key      []byte // partial key
		children [16]node
		value    []byte
	}
	leaf struct {
		key   []byte // partial key
		value []byte
	}
)

func (b *branch) childrenBitmap() uint16 {
	var bitmap uint16
	var i uint
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}

func (b *branch) Encode() ([]byte, error) {
	return nil, nil
}

func (l *leaf) Encode() ([]byte, error) {
	return nil, nil
}
