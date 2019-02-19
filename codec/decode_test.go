package codec

import (
	"bytes"
	"testing"
)

type decodeIntTest struct {
	val    []byte
	output int64
	error  string
}

type decodeByteArrayTest struct {
	val    []byte
	output []byte
	error  string
}

var decodeIntTests = []decodeIntTest{
	// compact integers
	{val: []byte{0x00}, output: int64(0)},
	{val: []byte{0x04}, output: int64(1)},
	{val: []byte{0xa8}, output: int64(42)},
	{val: []byte{0x01, 0x01}, output: int64(64)},
	{val: []byte{0x15, 0x01}, output: int64(69)},
	{val: []byte{0xfd, 0xff}, output: int64(16383)},
	{val: []byte{0x02, 0x00, 0x01, 0x00}, output: int64(16384)},
	{val: []byte{0xfe, 0xff, 0xff, 0xff}, output: int64(1073741823)},
	{val: []byte{0x03, 0x00, 0x00, 0x00, 0x40}, output: int64(1073741824)},
	{val: []byte{0x03, 0xff, 0xff, 0xff, 0xff}, output: int64(1<<32 - 1)},
	{val: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01}, output: int64(1 << 32)},
}

var decodeByteArrayTests = []decodeByteArrayTest{
	// byte arrays
	{val: []byte{0x04, 0x01}, output: []byte{0x01}},
	{val: []byte{0x04, 0xff}, output: []byte{0xff}},
	{val: []byte{0x08, 0x01, 0x01}, output: []byte{0x01, 0x01}},
	{val: append([]byte{0x01, 0x01}, byteArray(64)...), output: byteArray(64)},
	{val: append([]byte{0xfd, 0xff}, byteArray(16384)...), output: byteArray(16383)},
	{val: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...), output: byteArray(16384)},
}

func TestDecodeInts(t *testing.T) {
	for _, test := range decodeIntTests {
		output, err := DecodeInteger(test.val)
		if err != nil {
			t.Error(err)
		} else if output != test.output {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		} 
	}
}

func TestDecodeByteArrays(t *testing.T) {
	for _, test := range decodeByteArrayTests {
		output, err := DecodeByteArray(test.val)
		if err != nil {
			t.Error(err)
		} else if !bytes.Equal(output, test.output) {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		} 
	}
}
