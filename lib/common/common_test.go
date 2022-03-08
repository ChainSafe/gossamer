// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToInts(t *testing.T) {
	in := "1,2,3,4,-1"
	expected := []int{1, 2, 3, 4, -1}
	res, err := StringToInts(in)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("Fail: got %v expected %v", res, expected)
	}

	in = "17"
	expected = []int{17}
	res, err = StringToInts(in)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("Fail: got %v expected %v", res, expected)
	}

	in = "1,noot"
	_, err = StringToInts(in)
	if err == nil {
		t.Fatal("should fail")
	}
}

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		in  string
		out []byte
	}{
		{"0x0fc1", []byte{0x0f, 0xc1}},
		{"0x00", []byte{0x0}},
	}

	for _, test := range tests {
		res, err := HexToBytes(test.in)
		if err != nil {
			t.Errorf("Fail: error %s", err)
		} else if !bytes.Equal(res, test.out) {
			t.Errorf("Fail: got %x expected %x", res, test.out)
		}
	}
}

func TestHexToBytesFailing(t *testing.T) {
	_, err := HexToBytes("1234")
	if err == nil {
		t.Error("Fail: should error")
	}
}

func TestHexToHash(t *testing.T) {
	tests := []struct {
		in  string
		out []byte
	}{
		{
			in: "0x8550326cee1e1b768a254095b412e0db58523c2b5df9b7d2540b4513d475ce7f",
			out: []byte{
				0x85, 0x50, 0x32, 0x6c, 0xee, 0x1e, 0x1b, 0x76, 0x8a, 0x25, 0x40,
				0x95, 0xb4, 0x12, 0xe0, 0xdb, 0x58, 0x52, 0x3c, 0x2b, 0x5d, 0xf9,
				0xb7, 0xd2, 0x54, 0x0b, 0x45, 0x13, 0xd4, 0x75, 0xce, 0x7f}},
		{
			in: "0x00",
			out: []byte{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			}},
		{
			in: "0x8550326cee1e1b768a254095b412e0db58523c2b5df9b7d2540b4513d475ce7f00",
			out: []byte{
				0x85, 0x50, 0x32, 0x6c, 0xee, 0x1e, 0x1b, 0x76, 0x8a, 0x25, 0x40,
				0x95, 0xb4, 0x12, 0xe0, 0xdb, 0x58, 0x52, 0x3c, 0x2b, 0x5d, 0xf9,
				0xb7, 0xd2, 0x54, 0x0b, 0x45, 0x13, 0xd4, 0x75, 0xce, 0x7f}},
	}

	for _, test := range tests {
		res, err := HexToHash(test.in)
		byteRes := [32]byte(res)
		if err != nil {
			t.Errorf("Fail: error %s", err)
		} else if !bytes.Equal(byteRes[:], test.out) {
			t.Errorf("Fail: got %x expected %x", res, test.out)
		}
	}
}

type concatTest struct {
	a, b   []byte
	output []byte
}

var concatTests = []concatTest{
	{a: []byte{}, b: []byte{}, output: []byte{}},
	{a: []byte{0x00}, b: []byte{}, output: []byte{0x00}},
	{a: []byte{0x00}, b: []byte{0x00}, output: []byte{0x00, 0x00}},
	{a: []byte{0x00}, b: []byte{0x00, 0x01}, output: []byte{0x00, 0x00, 0x01}},
	{a: []byte{0x01}, b: []byte{0x00, 0x01, 0x02}, output: []byte{0x01, 0x00, 0x01, 0x02}},
	{
		a:      []byte{0x00, 0x01, 0x02, 0x00},
		b:      []byte{0x00, 0x01, 0x02},
		output: []byte{0x000, 0x01, 0x02, 0x00, 0x00, 0x01, 0x02}},
}

func TestConcat(t *testing.T) {
	for _, test := range concatTests {
		output := Concat(test.a, test.b...)
		if !bytes.Equal(output, test.output) {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		}
	}
}

func Test_UintToBytes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		n uint
		b []byte
	}{
		"zero": {
			n: 0,
			b: []byte{},
		},
		"one": {
			n: 1,
			b: []byte{1},
		},
		"256": {
			n: 256,
			b: []byte{1, 0},
		},
		"max uint32": {
			n: 1<<32 - 1,
			b: []byte{255, 255, 255, 255},
		},
		"one plus max uint32": {
			n: 1 + (1<<32 - 1),
			b: []byte{1, 0, 0, 0, 0},
		},
		"max int64": {
			n: 1<<63 - 1,
			b: []byte{0x7f, 255, 255, 255, 255, 255, 255, 255},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			b := UintToBytes(testCase.n)

			assert.Equal(t, testCase.b, b)

			bigIntBytes := big.NewInt(int64(testCase.n)).Bytes()
			assert.Equal(t, bigIntBytes, b)
		})
	}
}

func Test_BytesToUint(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		b []byte
		n uint
	}{
		"zero": {
			b: []byte{},
			n: 0,
		},
		"one": {
			b: []byte{1},
			n: 1,
		},
		"256": {
			b: []byte{1, 0},
			n: 256,
		},
		"max uint32": {
			b: []byte{255, 255, 255, 255},
			n: 1<<32 - 1,
		},
		"one plus max uint32": {
			b: []byte{1, 0, 0, 0, 0},
			n: 1 + (1<<32 - 1),
		},
		"max int64": {
			b: []byte{0x7f, 255, 255, 255, 255, 255, 255, 255},
			n: 1<<63 - 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			n := BytesToUint(testCase.b)

			assert.Equal(t, testCase.n, n)

			bigIntUint := uint(big.NewInt(0).SetBytes(testCase.b).Uint64())
			assert.Equal(t, bigIntUint, n)
		})
	}
}

func TestUint16ToBytes(t *testing.T) {
	tests := []struct {
		input    uint16
		expected []byte
	}{
		{uint16(0), []byte{0x0, 0x0}},
		{uint16(1), []byte{0x1, 0x0}},
		{uint16(255), []byte{0xff, 0x0}},
	}

	for _, test := range tests {
		res := Uint16ToBytes(test.input)
		if !bytes.Equal(res, test.expected) {
			t.Errorf("Output doesn't match expected. got=%v expected=%v\n", res, test.expected)
		}
	}
}

func TestSwapByteNibbles(t *testing.T) {
	tests := []struct {
		input    byte
		expected byte
	}{
		{byte(0xA0), byte(0x0A)},
		{byte(0), byte(0)},
		{byte(0x24), byte(0x42)},
	}

	for _, test := range tests {
		res := SwapByteNibbles(test.input)
		if res != test.expected {
			t.Fatalf("got: %x; expected: %x", res, test.expected)
		}
	}
}

func TestSwapNibbles(t *testing.T) {
	tests := []struct {
		key        []byte
		encodedKey []byte
	}{
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x05}, []byte{0x10, 0x20, 0x30, 0x40, 0x50}},
		{[]byte{0xff, 0x0, 0xAA, 0x81}, []byte{0xff, 0x00, 0xAA, 0x18}},
		{[]byte{0xAC, 0x19, 0x15}, []byte{0xCA, 0x91, 0x51}},
	}

	for _, test := range tests {
		res := SwapNibbles(test.key)
		if !bytes.Equal(res, test.encodedKey) {
			t.Fatalf("got: %x, expected: %x", res, test.encodedKey)
		}

		res = SwapNibbles(res)
		if !bytes.Equal(res, test.key) {
			t.Fatalf("Re-encoding failed. got: %x expected: %x", res, test.key)
		}
	}
}
