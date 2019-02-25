package hexcodec

import (
	"testing"
)

func TestHexEncode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte{0xF, 0x0}, []byte{0x0, 0x0F}},
		{[]byte{0xA, 0x8, 0xF, 0x0}, []byte{0x0, 0x8A, 0x0F}},
		{[]byte{0xD, 0xE, 0xA, 0xD, 0xB, 0xE, 0xE, 0xF}, []byte{0x0, 0xED, 0xDA, 0xEB, 0xEF}},
		{[]byte{0xF}, []byte{0xF}},
		{[]byte{0xA, 0xF, 0x0}, []byte{0xA, 0x0F}},
		{[]byte{0xA, 0xC, 0xA, 0xB, 0x1, 0x2, 0x3}, []byte{0xA, 0xAC, 0x1B, 0x32}},
	}

	for _, test := range tests {
		res := Encode(test.input)
		for i := 0; i < len(res); i++ {
			if res[i] != test.expected[i] {
				t.Fatalf("Output doesn't match expected. got=%v expected=%v\n", res, test.expected)
			}
		}
	}
}
