package trie

import (
	"testing"
)

// make byte array with length specified; used to test byte array encoding
func byteArray(length int) []byte {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = 0xff
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
		br        *branch
		header byte
	}{
		{&branch{nil, [16]node{}, nil}, byte(2)},
		{&branch{[]byte{0x00}, [16]node{}, nil}, byte(6)},
		{&branch{[]byte{0x00, 0x00, 0xf, 0x3}, [16]node{}, nil}, byte(18)},
		{&branch{nil, [16]node{}, []byte{0x01}}, byte(3)},
		{&branch{[]byte{0x00}, [16]node{}, []byte{0x01}}, byte(7)},
		{&branch{[]byte{0x00, 0x00}, [16]node{}, []byte{0x01}}, byte(11)},
		{&branch{[]byte{0x00, 0x00, 0xf}, [16]node{}, []byte{0x01}}, byte(15)},
		{&branch{byteArray(62), [16]node{}, nil}, 0xfa},
		{&branch{byteArray(62), [16]node{}, []byte{0x00}}, 0xfb},
		{&branch{byteArray(63), [16]node{}, nil}, byte(254)},
		{&branch{byteArray(64), [16]node{}, nil}, byte(254)},
		{&branch{byteArray(64), [16]node{}, []byte{0x01}}, byte(255)},
	}

	for _, test := range tests {
		res := test.br.header()
		if res != test.header {
			t.Errorf("Branch header fail: got %x expected %x", res, test.header)
		}
	}
}

func TestLeafHeader(t *testing.T) {
	tests := []struct {
		br        *leaf
		header byte
	}{
		{&leaf{nil, nil}, byte(1)},
		{&leaf{[]byte{0x00}, nil}, byte(5)},
		{&leaf{[]byte{0x00, 0x00, 0xf, 0x3}, nil}, byte(17)},
		{&leaf{byteArray(62), nil}, 0xf9},
		{&leaf{byteArray(63), nil}, byte(253)},
		{&leaf{byteArray(64), []byte{0x01}}, byte(253)},
	}

	for _, test := range tests {
		res := test.br.header()
		if res != test.header {
			t.Errorf("Branch header fail: got %x expected %x", res, test.header)
		}
	}
}