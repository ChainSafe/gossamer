package codec

import (
	"bytes"
	"strings"
	"testing"
)

type encodeTest struct {
	val    interface{}
	output []byte
}

var encodeTests = []encodeTest{
	// compact integers
	{val: int64(0), output: []byte{0x00}},
	{val: int64(1), output: []byte{0x04}},
	{val: int64(42), output: []byte{0xa8}},
	{val: int64(69), output: []byte{0x15, 0x01}},
	{val: int64(16383), output: []byte{0xfd, 0xff}},
	{val: int64(16384), output: []byte{0x02, 0x00, 0x01, 0x00}},
	{val: int64(1073741823), output: []byte{0xfe, 0xff, 0xff, 0xff}},
	{val: int64(1073741824), output: []byte{0x03, 0x00, 0x00, 0x00, 0x40}},
	{val: int64(1<<32 - 1), output: []byte{0x03, 0xff, 0xff, 0xff, 0xff}},
	{val: int64(1 << 32), output: []byte{0x07, 0x00, 0x00, 0x00, 0x00, 0x01}},

	// byte arrays
	{val: []byte{0x01}, output: []byte{0x04, 0x01}},
	{val: []byte{0xff}, output: []byte{0x04, 0xff}},
	{val: []byte{0x01, 0x01}, output: []byte{0x08, 0x01, 0x01}},
	{val: []byte{0x01, 0x01}, output: []byte{0x08, 0x01, 0x01}},
	{val: byteArray(64), output: append([]byte{0x01, 0x01}, byteArray(64)...)},
	{val: byteArray(16384), output: append([]byte{0x02, 0x00, 0x01, 0x00}, byteArray(16384)...)},

	// booleans
	{val: true, output: []byte{0x01}},
	{val: false, output: []byte{0x00}},
}

func TestEncode(t *testing.T) {

	// Test strings for various values of n, mode. Also test strings with special characters
	// TODO: Unicode?
	testString1 := "We love you! We believe in open source as wonderful form of giving."
	testString2 := strings.Repeat("We need a longer string to test with. Let's multiple this several times.", 230) //n = 16560
	//testString3 := strings.Repeat("We need a longer string to test with. Let's multiple this several times.", 14913081) //n = 1 073 741 832 (> 2^30 = 1 073 741 824)

	var testStrings = []encodeTest{
		{val: string("a"),
		 output: []byte{0x04, 0x61}},
		{val: string("go-pre"),																// n =  6 = 0b110, mode = 0 = 0b00
		 output: append([]byte{0x18}, string("go-pre")...)}, 							 	// n|mode = 0b11000 = 0x18, Enc = n | mode | 0x("go-pre")
		{val: testString1,																	// n = 67 = 0b1000011, mode = 1 = 0b01
		 output: append([]byte{0x0D,0x01}, testString1...)}, 								// n|mode = 0b00000001 00001101 = 0x010D (big endian) = 0x0D01 (little endian), Enc = n | mode | 0x("We love you!...")
		{val: testString2,																	// n = 16560 = 0b1000000 10110000, mode = 2 = 0b10
		 output: append([]byte{0xC2,0x02,0x01,0x00}, testString2...)},					 	// n|mode = 0b 00000001 00000010 11000010 = 0x102C2 (big endian) = 0xC20201 (little endian), Enc = n | mode | 0x("We need a...")
		//{val: testString3,																	// n = 14913081 = 11100011 10001110 00111001, mode = 3 = 0b11, num_bytes_n = 3
		//output: append([]byte{0x08,0x00,0x00,0x40,0x03}, testString3...)},					// (num_bytes_n - 4)|mode|n = 0b 00000011 01000000 00000000 00000000 00001000 = 0x03 40 00 00 08 (big endian) = 0x08 00 00 40 03 (little endian), Enc = (num_bytes_n - 4)|mode|n| 0x("We need a...")
	}

	//Append stringTests to other encodeTests
	for _, testString := range testStrings {
		encodeTests = append(encodeTests, testString)
	}

	//Run all tests
	for _, test := range encodeTests {
		output, err := Encode(test.val)
		if err != nil {
			t.Error(err)
		} else if !bytes.Equal(output, test.output) {
			t.Errorf("Fail: got %x expected %x", output, test.output)
		}
	}
}
