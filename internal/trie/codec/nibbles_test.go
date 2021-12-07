// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NibblesToKeyLE(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		nibbles []byte
		keyLE   []byte
	}{
		"nil nibbles": {
			keyLE: []byte{},
		},
		"empty nibbles": {
			nibbles: []byte{},
			keyLE:   []byte{},
		},
		"0xF 0xF": {
			nibbles: []byte{0xF, 0xF},
			keyLE:   []byte{0xFF},
		},
		"0x3 0xa 0x0 0x5": {
			nibbles: []byte{0x3, 0xa, 0x0, 0x5},
			keyLE:   []byte{0x3a, 0x05},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1},
			keyLE:   []byte{0xaa, 0xff, 0x01},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1 0xc 0x2": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x2},
			keyLE:   []byte{0xaa, 0xff, 0x01, 0xc2},
		},
		"0xa 0xa 0xf 0xf 0x0 0x1 0xc": {
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc},
			keyLE:   []byte{0xa, 0xaf, 0xf0, 0x1c},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keyLE := NibblesToKeyLE(testCase.nibbles)

			assert.Equal(t, testCase.keyLE, keyLE)
		})
	}
}

func Test_KeyLEToNibbles(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		in      []byte
		nibbles []byte
	}{
		"nil input": {
			nibbles: []byte{},
		},
		"empty input": {
			in:      []byte{},
			nibbles: []byte{},
		},
		"0x0": {
			in:      []byte{0x0},
			nibbles: []byte{0, 0}},
		"0xFF": {
			in:      []byte{0xFF},
			nibbles: []byte{0xF, 0xF}},
		"0x3a 0x05": {
			in:      []byte{0x3a, 0x05},
			nibbles: []byte{0x3, 0xa, 0x0, 0x5}},
		"0xAA 0xFF 0x01": {
			in:      []byte{0xAA, 0xFF, 0x01},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1}},
		"0xAA 0xFF 0x01 0xc2": {
			in:      []byte{0xAA, 0xFF, 0x01, 0xc2},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x2}},
		"0xAA 0xFF 0x01 0xc0": {
			in:      []byte{0xAA, 0xFF, 0x01, 0xc0},
			nibbles: []byte{0xa, 0xa, 0xf, 0xf, 0x0, 0x1, 0xc, 0x0}},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nibbles := KeyLEToNibbles(testCase.in)

			assert.Equal(t, testCase.nibbles, nibbles)
		})
	}
}

func Test_NibblesKeyLE(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		nibblesToEncode []byte
		nibblesDecoded  []byte
	}{
		"empty input": {
			nibblesToEncode: []byte{},
			nibblesDecoded:  []byte{},
		},
		"one byte": {
			nibblesToEncode: []byte{1},
			nibblesDecoded:  []byte{0, 1},
		},
		"two bytes": {
			nibblesToEncode: []byte{1, 2},
			nibblesDecoded:  []byte{1, 2},
		},
		"three bytes": {
			nibblesToEncode: []byte{1, 2, 3},
			nibblesDecoded:  []byte{0, 1, 2, 3},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keyLE := NibblesToKeyLE(testCase.nibblesToEncode)
			nibblesDecoded := KeyLEToNibbles(keyLE)

			assert.Equal(t, testCase.nibblesDecoded, nibblesDecoded)
		})
	}
}
