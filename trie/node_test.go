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

package trie

import (
	"bytes"
	"testing"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
)

// byteArray makes byte array with length specified; used to test byte array encoding
func byteArray(length int) []byte {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = 0xf
	}
	return b
}

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

func TestBranchHeader(t *testing.T) {
	tests := []struct {
		br     *branch
		header []byte
	}{
		{&branch{nil, [16]node{}, nil, true}, []byte{2}},
		{&branch{[]byte{0x00}, [16]node{}, nil, true}, []byte{6}},
		{&branch{[]byte{0x00, 0x00, 0xf, 0x3}, [16]node{}, nil, true}, []byte{18}},

		{&branch{nil, [16]node{}, []byte{0x01}, true}, []byte{3}},
		{&branch{[]byte{0x00}, [16]node{}, []byte{0x01}, true}, []byte{7}},
		{&branch{[]byte{0x00, 0x00}, [16]node{}, []byte{0x01}, true}, []byte{11}},
		{&branch{[]byte{0x00, 0x00, 0xf}, [16]node{}, []byte{0x01}, true}, []byte{15}},

		{&branch{byteArray(62), [16]node{}, nil, true}, []byte{0xfa}},
		{&branch{byteArray(62), [16]node{}, []byte{0x00}, true}, []byte{0xfb}},
		{&branch{byteArray(63), [16]node{}, nil, true}, []byte{254, 0}},
		{&branch{byteArray(64), [16]node{}, nil, true}, []byte{254, 1}},
		{&branch{byteArray(64), [16]node{}, []byte{0x01}, true}, []byte{255, 1}},

		{&branch{byteArray(317), [16]node{}, []byte{0x01}, true}, []byte{255, 254}},
		{&branch{byteArray(318), [16]node{}, []byte{0x01}, true}, []byte{255, 255, 0}},
		{&branch{byteArray(573), [16]node{}, []byte{0x01}, true}, []byte{255, 255, 255, 0}},
	}

	for _, test := range tests {
		res := test.br.header()
		if !bytes.Equal(res, test.header) {
			t.Errorf("Branch header fail case %v: got %x expected %x", test.br, res, test.header)
		}
	}
}

func TestLeafHeader(t *testing.T) {
	tests := []struct {
		br     *leaf
		header []byte
	}{
		{&leaf{nil, nil, true}, []byte{1}},
		{&leaf{[]byte{0x00}, nil, true}, []byte{5}},
		{&leaf{[]byte{0x00, 0x00, 0xf, 0x3}, nil, true}, []byte{17}},
		{&leaf{byteArray(62), nil, true}, []byte{0xf9}},
		{&leaf{byteArray(63), nil, true}, []byte{253, 0}},
		{&leaf{byteArray(64), []byte{0x01}, true}, []byte{253, 1}},

		{&leaf{byteArray(318), []byte{0x01}, true}, []byte{253, 255, 0}},
		{&leaf{byteArray(573), []byte{0x01}, true}, []byte{253, 255, 255, 0}},
	}

	for _, test := range tests {
		res := test.br.header()
		if !bytes.Equal(res, test.header) {
			t.Errorf("Leaf header fail: got %x expected %x", res, test.header)
		}
	}
}

func TestBranchEncode(t *testing.T) {
	randKeys := generateRand(100)
	randVals := generateRand(100)

	for i, testKey := range randKeys {
		b := &branch{key: testKey, children: [16]node{}, value: randVals[i]}
		expected := []byte{}

		expected = append(expected, b.header()...)
		expected = append(expected, nibblesToKey(b.key)...)

		expected = append(expected, common.Uint16ToBytes(b.childrenBitmap())...)

		for _, child := range b.children {
			if child != nil {
				encChild, err := Encode(child)
				if err != nil {
					t.Errorf("Fail when encoding branch child: %s", err)
				}
				expected = append(expected, encChild...)
			}
		}

		buf := bytes.Buffer{}
		encoder := &scale.Encoder{Writer: &buf}
		_, err := encoder.Encode(b.value)
		if err != nil {
			t.Fatalf("Fail when encoding value with scale: %s", err)
		}

		expected = append(expected, buf.Bytes()...)

		res, err := b.Encode()
		if !bytes.Equal(res, expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}
}

func TestLeafEncode(t *testing.T) {
	randKeys := generateRand(100)
	randVals := generateRand(100)

	for i, testKey := range randKeys {
		l := &leaf{key: testKey, value: randVals[i]}
		expected := []byte{}

		expected = append(expected, l.header()...)
		expected = append(expected, nibblesToKey(l.key)...)

		buf := bytes.Buffer{}
		encoder := &scale.Encoder{Writer: &buf}
		_, err := encoder.Encode(l.value)
		if err != nil {
			t.Fatalf("Fail when encoding value with scale: %s", err)
		}

		expected = append(expected, buf.Bytes()...)

		res, err := l.Encode()
		if !bytes.Equal(res, expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}
}

func TestEncodeRoot(t *testing.T) {
	trie := newEmpty()

	for i := 0; i < 20; i++ {
		rt := generateRandTest(16)
		for _, test := range rt {
			err := trie.Put(test.key, test.value)
			if err != nil {
				t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}

			_, err = Encode(trie.root)
			if err != nil {
				t.Errorf("Fail to encode trie root: %s", err)
			}
		}
	}
}
