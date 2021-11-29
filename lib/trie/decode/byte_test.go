// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package decode

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ReadNextByte(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		reader     io.Reader
		b          byte
		errWrapped error
		errMessage string
	}{
		"empty buffer": {
			reader:     bytes.NewBuffer(nil),
			errWrapped: io.EOF,
			errMessage: "EOF",
		},
		"single byte buffer": {
			reader: bytes.NewBuffer([]byte{1}),
			b:      1,
		},
		"two bytes buffer": {
			reader: bytes.NewBuffer([]byte{1, 2}),
			b:      1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			b, err := ReadNextByte(testCase.reader)

			assert.ErrorIs(t, err, testCase.errWrapped)
			if err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}
			assert.Equal(t, testCase.b, b)
		})
	}
}
