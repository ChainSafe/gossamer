package trie

import (
	"bytes"
	"testing"
)

func TestKeyEncodeByte(t *testing.T) {
	tests := []struct{
		input byte
		expected byte
	}{
		{byte(36), byte(66)},
		{byte(0), byte(0)},
		{byte(240), byte(15)},
	}

	for _, test := range tests {
		res := keyEncodeByte(test.input)
		if res != test.expected {
			t.Fatalf("got: %x; expected: %x", res, test.expected)
		}
	}
}

func TestKeyEncode(t *testing.T) {
	tests := []struct{
		key	[]byte
		encodedKey []byte
	}{
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x05},[]byte{0x10, 0x20, 0x30, 0x40, 0x50}},
		{[]byte{0xff, 0x0, 0xAA, 0x81}, []byte{0xff, 0x00, 0xAA, 0x18}},
		{[]byte{0xAC, 0x19, 0x15}, []byte{0xCA, 0x91, 0x51}},
	}

	for _, test := range tests {
		res := KeyEncode(test.key)
		if !bytes.Equal(res, test.encodedKey) {
			t.Fatalf("got: %x, expected: %x", res, test.encodedKey)
		}

		res = KeyEncode(res)
		if !bytes.Equal(res, test.key) {
			t.Fatalf("Re-encoding failed. got: %x expected: %x", res, test.key)
		}
	}

}