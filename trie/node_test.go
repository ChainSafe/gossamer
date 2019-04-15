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

import (
	"testing"
)

func TestChildrenBitmap(t *testing.T) {
	b := &branch{children: [16]node{}}
	res := b.childrenBitmap()
	if res != 0 {
		t.Errorf("Fail to get children bitmap: got %x expected %x", res, 1)
	}

	b.children[0] = &leaf{key: []byte{0x00}, value: []byte{0x00}}
	res = b.childrenBitmap()
	if res != 1 {
		t.Errorf("Fail to get children bitmap: got %x expected %x", res, 1)
	}

	b.children[4] = &leaf{key: []byte{0x00}, value: []byte{0x00}}
	res = b.childrenBitmap()
	if res != 1<<4+1 {
		t.Errorf("Fail to get children bitmap: got %x expected %x", res, 17)
	}

	b.children[15] = &leaf{key: []byte{0x00}, value: []byte{0x00}}
	res = b.childrenBitmap()
	if res != 1<<15+1<<4+1 {
		t.Errorf("Fail to get children bitmap: got %x expected %x", res, 257)
	}
}
